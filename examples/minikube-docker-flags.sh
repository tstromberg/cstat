#!/bin/bash
#
# Gather data comparing different flags for minikube+Docker
readonly TESTS=$1

# How many iterations to cycle through
readonly TEST_ITERATIONS=4

# How long to poll CPU usage for (each point is an average over this period)
readonly POLL_DURATION=5s

# How long to measure background usage for. 
readonly TOTAL_DURATION=25s

# How all tests will be identified
readonly SESSION_ID="$(date +%Y%m%d-%H%M%S)-$$"

measure() {
  local name=$1
  local iteration=$2
  
  sanitized=${name// /_}
  local filename="results/${SESSION_ID}/cstat.${sanitized}.$$-${iteration}"
  echo "filename: ${filename}"

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
    apps=$(osascript -e 'tell application "System Events" to get name of (processes where background only is false)'  | tr ',' '\n' | sed s/"^ "//g)
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
  minikube version || { echo "minikube version failed"; exit 1; }
  docker version
  echo "----------------------------------------------------"
  echo ""

  echo "  >> Starting Docker for Desktop ..."
  open -a Docker
  docker ps || sleep 15
  docker ps || sleep 15
  docker ps || sleep 15

  kubectl --context docker-desktop version
  if [[ $? == 0 ]]; then
    echo "Kubernetes is running in Docker for Desktop - please stop it"
    exit 2
  fi

  for i in $(seq 1 ${TEST_ITERATIONS}); do
    echo ""
    echo "==> session ${SESSION_ID}, iteration $i"
    cleanup

    flags="--wait=all --driver=docker --extra-config controller-manager.leader-elect=false --extra-config scheduler.leader-elect=false"
    time minikube start $flags && measure "${flags}" $i
    cleanup

    flags="--wait=all --driver=docker --extra-config controller-manager.leader-elect=false"
    time minikube start $flags && measure "${flags}" $i
    cleanup

    flags="--wait=all --driver=docker --extra-config scheduler.leader-elect=false"
    time minikube start $flags && measure "${flags}" $i
    cleanup

    flags="--wait=all --driver=docker"
    time minikube start $flags && measure "${flags}" $i
    cleanup

    # Measure the background noise on this system
    sleep 5
    measure docker $i
  done
}

main "$@"
