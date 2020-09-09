#!/bin/bash
#
# Gather data comparing the overhead of multiple local Kubernetes (macOS only)
readonly TESTS=$1

# How many iterations to cycle through
readonly TEST_ITERATIONS=10

# How long to poll CPU usage for (each point is an average over this period)
readonly POLL_DURATION=5s

# How long to measure background usage for. 5 minutes too short, 10 minutes too long
readonly TOTAL_DURATION=5m

# How all tests will be identified
readonly SESSION_ID="$(date +%Y%m%d-%H%M%S)-$$"

measure() {
  local name=$1
  local iteration=$2
  local filename="results/${SESSION_ID}/cstat.${name}.$$-${iteration}"

  echo ""
  echo "  >> Current top processes by CPU:"
  top -n 3 -l 2 -s 2 -o cpu  | tail -n4 | awk '{ print $1 " " $2 " " $3 " " $4 }'

  if [[ "${iteration}" == 0 ]]; then
    echo "NOTE: dry-run iteration: will not record measurements"
    cstat --poll "${POLL_DURATION}" --for "${POLL_DURATION}" --busy
    return
  fi

  echo ""
  echo "  >> Measuring ${name} and saving to ${filename} ..."
  cstat --poll "${POLL_DURATION}" --for "${TOTAL_DURATION}" --busy --header=false | tee "${filename}"
}


cleanup() {
  echo "  >> Deleting local clusters and Docker containers ..."
  minikube delete --all 2>/dev/null >/dev/null
  k3d cluster delete 2>/dev/null >/dev/null
  kind delete cluster 2>/dev/null >/dev/null
  docker stop $(docker ps -q) 2>/dev/null
  docker kill $(docker ps -q) 2>/dev/null
  docker rm $(docker ps -a -q) 2>/dev/null
  sleep 2
}

pause_if_running_apps() {
  while true; do
    local apps=$(osascript -e 'tell application "System Events" to get name of (processes where background only is false)'  | tr ',' '\n' | sed s/"^ "//g)
    local quiet=0

    for app in $apps; do
      quiet=1
      if [[ "${app}" != "Terminal" && "${app}" != "Finder" ]]; then
        echo "Unexpected application running: \"${app}\" - will sleep"
        quiet=0
      fi
    done

    pmset -g batt | grep 'AC Power'
    if [[ "$?" != 0 ]]; then
      echo "waiting to be plugged in ..."
      sleep 5
      continue
    fi

    if [[ "${quiet}" == 1 ]]; then
      break
    else
      echo "waiting for apps to be closed ..."
      sleep 5
    fi

  done
}

fail() {
  local name=$1
  local iteration=$2

  echo '***********************************************************************'
  echo "${name} failed on iteration ${iteration} - will not record measurement"
  echo '***********************************************************************'

  if [[ "${iteration}" == 0 ]]; then
    echo "test environment appears invalid, exiting"
    exit 90
  fi
}

start_docker() {
    local docker_up=0
    local started=0

    while [[ "${docker_up}" == 0 ]]; do
      docker info >/dev/null && docker_up=1 || docker_up=0

      if [[ "${docker_up}" == 0 && "${started}" == 0 ]]; then
          echo ""
          echo "  >> Starting Docker for Desktop ..."
          open -a Docker
          started=1
      fi

      sleep 1
    done

    # Give time for d4d Kubernetes to begin, if it's around
    if [[ "${started}" == 1 ]]; then
      sleep 15
    fi
}


main() {
  echo "Session ID: ${SESSION_ID}"
  mkdir -p "results/${SESSION_ID}"

  echo "----[ versions ]------------------------------------"
  k3d version || { echo "k3d version failed"; exit 1; }
  kind version || { echo "kind version failed"; exit 1; }
  minikube version || { echo "minikube version failed"; exit 1; }
  docker version
  echo "----------------------------------------------------"
  echo ""

  echo "Turning on Wi-Fi for initial downloads"
  networksetup -setairportpower Wi-Fi on

  for i in $(seq 0 ${TEST_ITERATIONS}); do
    echo ""
    echo "==> session ${SESSION_ID}, iteration $i"


    cleanup

    if [[ "$i" = 0 ]]; then
      echo "NOTE: The 0 iteration is an unmeasured dry run!"
    else
      pause_if_running_apps
      echo "Turning off Wi-Fi to remove background noise"
      networksetup -setairportpower Wi-Fi off

      echo "  >> Killing Docker for Desktop ..."
      osascript -e 'quit app "Docker"'

      # Measure the background noise on this system
      sleep 15
      measure idle $i
    fi

    # Run cleanup once we can assert that Docker is up
    start_docker
    cleanup

    docker_k8s=0
    kubectl --context docker-desktop version
    if  [[ $? == 0 ]]; then
      echo "Kubernetes is running in Docker for Desktop - adjusting tests"
      docker_k8s=1
      measure docker_k8s $i
    else
      measure docker $i
    fi

    if [[ "${docker_k8s}" == 0 ]]; then
      echo ""
      echo "-> k3d"
      time k3d cluster create && measure k3d $i || fail k3d $i
      cleanup

      echo ""
      echo "-> kind"
      time kind create cluster && measure kind $i || fail kind $i
      cleanup
    fi

    # test different drivers
    for driver in docker hyperkit; do
      if [[ "${docker_k8s}" == 1 && "${driver}" == "docker" ]]; then
        echo "  >> Quitting Docker for Desktop ..."
        osascript -e 'quit app "Docker"'
        continue
      fi

      echo ""
      echo "-> minikube --driver=${driver}"
      time minikube start --driver "${driver}" && measure "minikube.${driver}" $i || fail "minikube.${driver}" $i
      cleanup

      # We won't be needing docker for the remaining tests this iteration
      if [[ "${driver}" == "docker" ]]; then
        echo "  >> Quitting Docker for Desktop ..."
        osascript -e 'quit app "Docker"'
      fi
    done ## driver
  done ## iteration
}

main "$@"
