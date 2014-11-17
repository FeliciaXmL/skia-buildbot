#!/bin/bash
#
# Updates the code running on skiamonitor.com.
#
set -x

source vm_config.sh

gcutil --project=$PROJECT_ID ssh --ssh_user=$PROJECT_USER $VM_NAME_BASE-monitoring \
  "cd ~/buildbot;" \
  "git pull;" \
  "cd compute_engine_scripts/monitoring;" \
  "./setup.sh"
