#!/bin/bash
#

# Arguments: Which versions of minikube to try
readonly VERSIONS=$*

# How many iterations to cycle through
readonly TEST_ITERATIONS=15

# How long to poll CPU usage for (each point is an average over this period)
readonly POLL_DURATION=5s

readonly TOTAL_DURATION=4m

# How all tests will be identified
readonly SESSION_ID="$(date +%Y%m%d-%H%M%S)-$$"

measure() {
  local name=$*
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

  minikube delete --all 2>/dev/null >/dev/null
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
  for version in ${VERSIONS}; do
    target="/tmp/minikube-${version}"
    echo "-> Downloading ${version} to ${target}"
    curl -L -C - -o "${target}" https://storage.googleapis.com/minikube/releases/${version}/minikube-darwin-amd64
    chmod 755 "${target}"
    "${target}" start --download-only
  done

  pause_if_running_apps
  echo "Session ID: ${SESSION_ID}"
  mkdir -p "results/${SESSION_ID}"

  for i in $(seq 1 ${TEST_ITERATIONS}); do
    echo ""
    echo "==> session ${SESSION_ID}, iteration $i"
    cleanup
    sleep 10
    echo "  >> Killing Docker for Desktop ..."
    osascript -e 'quit app "Docker"'

    target="/tmp/minikube-${version}"
    driver=hyperkit

   for version in ${VERSIONS}; do
      echo "-> minikube ${version} --driver=${driver}"
      time "${target}" start --driver "${driver}" -p "${driver}$$" && measure "minikube_hyperkit_${version}" $i
      cleanup
   done
 
  done ## iteration
}

main "$@"
