#!/bin/bash
#
# Gather data comparing the overhead of multiple local Kubernetes 
readonly TESTS=$1

# How many iterations to cycle through
readonly TEST_ITERATIONS=5

# How long to poll CPU usage for (each point is an average over this period)
readonly POLL_DURATION=5s

# How long to measure background usage for. 5 minutes too short, 10 minutes too long
readonly TOTAL_DURATION=7m

# How all tests will be identified
readonly SESSION_ID="$(date +%Y%m%d-%H%M%S)-$$"

measure() {
  local name=$1
  local iteration=$2
  local filename="results/${SESSION_ID}/cstat.${name}.$$-${iteration}"

  echo ""
  echo "  >> Current top processes by CPU:"
  top -n 3 -l 2 -s 2 -o cpu  | tail -n4 | awk '{ print $1 " " $2 " " $3 " " $4 }'

  echo ""
  echo "  >> Measuring ${name} and saving to ${filename} ..."
  cstat --poll "${POLL_DURATION}" --for "${TOTAL_DURATION}" --busy --header=false | tee "${filename}"
}


cleanup() {
  echo "  >> Deleting local clusters ..."

  # workaround delete hang w/ docker driver: https://github.com/kubernetes/minikube/issues/7657
  minikube unpause 2>/dev/null >/dev/null

  minikube delete --all 2>/dev/null >/dev/null
  k3d d 2>/dev/null >/dev/null
  kind delete cluster 2>/dev/null >/dev/null
  docker stop $(docker ps -q) 2>/dev/null
  docker kill $(docker ps -q) 2>/dev/null
  docker rm $(docker ps -a -q) 2>/dev/null

  sleep 5
  pause_if_running_apps
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


main() {
  pause_if_running_apps
  echo "Session ID: ${SESSION_ID}"
  mkdir -p "results/${SESSION_ID}"

  echo "----[ versions ]------------------------------------"
  k3d version || { echo "k3d version failed"; exit 1; }
  kind version || { echo "kind version failed"; exit 1; }
  minikube version || { echo "minikube version failed"; exit 1; }
  if [[ -x ./out/minikube ]]; then
    ./out/minikube version || { echo "./out/minikube version failed"; exit 1; }
  fi
  docker version
  echo "----------------------------------------------------"
  echo ""

  for i in $(seq 1 ${TEST_ITERATIONS}); do
    echo ""
    echo "==> session ${SESSION_ID}, iteration $i"
    cleanup
    echo "  >> Killing Docker for Desktop ..."
    osascript -e 'quit app "Docker"'

    # Measure the background noise on this system
    sleep 15
    measure idle $i

    echo ""
    echo "  >> Starting Docker for Desktop ..."
    open -a Docker

    # Sleep because we are too lazy to detect when Docker is up
    sleep 45
    # Run cleanup once more now that Docker is online
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
      time k3d c && measure k3d $i
      cleanup

      echo ""
      echo "-> kind"
      time kind create cluster && measure kind $i
      cleanup
    fi

    #  hyperkit virtualbox vmware
    for driver in docker hyperkit virtualbox vmware; do
      if [[ "${docker_k8s}" == 1 && "${driver}" == "docker" ]]; then
        echo "  >> Quitting Docker for Desktop ..."
        osascript -e 'quit app "Docker"'
        continue
      fi

      echo ""
      echo "-> minikube --driver=${driver}"
      time minikube start --driver "${driver}" && measure "minikube.${driver}" $i
      # minikube pause && measure "minikube_paused.${driver}" $i
      cleanup

      if [[ -x "./out/minikube" ]]; then
        echo "-> ./out/minikube --driver=${driver}"
        time ./out/minikube start --driver "${driver}" && measure "out.minikube.${driver}" $i
        # minikube pause && measure "out.minikube_paused.${driver}" $i
        cleanup
      fi

      # We won't be needing docker for the remaining tests this iteration
      if [[ "${driver}" == "docker" ]]; then
        echo "  >> Quitting Docker for Desktop ..."
        osascript -e 'quit app "Docker"'
      fi


    done ## driver
  done ## iteration
}

main "$@"
