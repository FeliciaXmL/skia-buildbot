#/bin/bash
# Creates the service account that has read-write access to the skottie bucket.

set -e -x

source ../kube/config.sh
source ../bash/ramdisk.sh

# New service account we will create.
SA_NAME=skia-skottie
cd /tmp/ramdisk
gcloud iam service-accounts create "${SA_NAME}" --display-name="Read-write access to GCS for skottie server."
gcloud beta iam service-accounts keys create ${SA_NAME}.json --iam-account="${SA_NAME}@${PROJECT_SUBDOMAIN}.iam.gserviceaccount.com"
gsutil acl ch -u "${SA_NAME}@${PROJECT_SUBDOMAIN}.iam.gserviceaccount.com:W" gs://skottie-renderer
kubectl create secret generic "${SA_NAME}" --from-file=key.json=${SA_NAME}.json
cd -
