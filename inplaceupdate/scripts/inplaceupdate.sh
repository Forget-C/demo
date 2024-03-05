#!/bin/bash
# 
name=${1}
namespace=${2}
container=${3}
image=${4}

usage="Usage: $0 <name> <namespace> <container> <image>"
# Check if the help command is used

if [ "${name}" == "help" ] && [ -z "${namespace}" ]; then
    echo ${usage}
    exit 1
fi 


if [ -z "${name}" ] || [ -z "${namespace}" ] || [ -z "${container}" ] || [ -z "${image}" ]; then
  echo ${usage}
  exit 1
fi

# Check deployment exists
kubectl get deployment ${name} -n ${namespace} &> /dev/null
if [ $? != 0 ]; then
  echo "Deployment ${name} does not exist in namespace ${namespace}"
  exit 1
fi

# Check container exists and image is different
kubectl get deployment ${name} -n ${namespace} -o jsonpath="{.spec.template.spec.containers[*].name}" | grep -w ${container} &> /dev/null
if [ $? != 0 ]; then
  echo "Container ${container} does not exist in deployment ${name}"
  exit 1
fi
kubectl get deployment ${name} -n ${namespace} -o jsonpath="{.spec.template.spec.containers[?(@.name==\"${container}\")].image}" | grep -w ${image} &> /dev/null
if [ $? == 0 ]; then
  echo "Image ${image} is already in deployment ${name}"
  exit 1
fi

# Check repicas are available
replicas=$(kubectl get deployment ${name} -n ${namespace} -o jsonpath="{.spec.replicas}")
available=$(kubectl get deployment ${name} -n ${namespace} -o jsonpath="{.status.availableReplicas}")
if [ "${replicas}" != "${available}" ]; then
  echo "Deployment ${name} is not available"
  echo "Replicas: ${replicas} - Available: ${available}"
  exit 1
fi

# get matchlabels from deployment
matchlabels=$(kubectl get deployment ${name} -n ${namespace} -o jsonpath="{.spec.selector.matchLabels}")
# {"name":"nginx"} --> name=nginx
matchlabels=$(echo ${matchlabels} | sed 's/{//g' | sed 's/}//g' | sed 's/\"//g' | sed 's/:/=/g')


# get other info for match rs
deploy_uid=$(kubectl get deployment ${name} -n ${namespace} -o jsonpath="{.metadata.uid}")

# get rs
rss=$(kubectl get rs -n ${namespace} -l ${matchlabels} -o jsonpath="{.items[*].metadata.name}")

for rs in ${rss}; do
  references=$(kubectl get rs ${rs} -n ${namespace} -o jsonpath="{.metadata.ownerReferences}")
  new_rs=""
  revision=0
  uid_check_pass=0
  for ref in $(jq -c '.[]' <<< ${references}); do
    uid=$(jq -r '.uid' <<< ${ref})
    if [ "${uid}" == "${deploy_uid}" ]; then
      uid_check_pass=1
      break
    fi
  done
  if [ ${uid_check_pass} -eq 1 ]; then
    this_revision=$(kubectl get rs ${rs} -n ${namespace} -o jsonpath="{.metadata.annotations.deployment\.kubernetes\.io/revision}")
    if [ ${this_revision} -gt ${revision} ]; then
      revision=${this_revision}
      new_rs=${rs}
    fi
  fi
done
rs=${new_rs}
rs_uid=$(kubectl get rs ${rs} -n ${namespace} -o jsonpath="{.metadata.uid}")

# check deployment is owned by rs
references=$(kubectl get rs ${rs} -n ${namespace} -o jsonpath="{.metadata.ownerReferences}")
uid_check_pass=0
for ref in $(jq -c '.[]' <<< ${references}); do
  uid=$(jq -r '.uid' <<< ${ref})
  if [ "${uid}" == "${deploy_uid}" ]; then
    uid_check_pass=1
    break
  fi
done
if [ ${uid_check_pass} -eq 0 ]; then
  echo "Replicaset ${rs} is not owned by deployment ${name}"
  exit 1
fi

# get pod
matchlabels=$(kubectl get rs ${rs} -n ${namespace} -o jsonpath="{.spec.selector.matchLabels}")
matchlabels=$(echo ${matchlabels} | sed 's/{//g' | sed 's/}//g' | sed 's/\"//g' | sed 's/:/=/g')
pod_names=$(kubectl get pod -n ${namespace} -l ${matchlabels} -o jsonpath="{.items[*].metadata.name}")

# pause deployment
kubectl rollout pause deployment ${name} -n ${namespace}

# change image for pods
# "pod1,0 pod2,0 pod3,0 "
pods_restart_count=""
for pod_name in ${pod_names}; do
    # check pod is owned by rs
    pod_ref=$(kubectl get pod ${pod_name} -n ${namespace} -o jsonpath="{.metadata.ownerReferences}")
    uid_check_pass=0
    for ref in $(jq -c '.[]' <<< ${pod_ref}); do
        uid=$(jq -r '.uid' <<< ${ref})
        if [ "${uid}" == "${rs_uid}" ]; then
            uid_check_pass=1
            break
        fi
    done
    if [ ${uid_check_pass} -eq 1 ]; then
        pod_cur_image=$(kubectl get pod ${pod_name} -n ${namespace} -o jsonpath="{.spec.containers[?(@.name==\"${container}\")].image}")
        if [ "${pod_cur_image}" == "${image}" ]; then
            echo "Pod ${pod_name} is already in ${image}"
            continue
        fi
        pod_restart_count=$(kubectl get pod ${pod_name} -n ${namespace} -o jsonpath="{.status.containerStatuses[?(@.name==\"${container}\")].restartCount}")
        kubectl set image pod ${pod_name} -n ${namespace} ${container}=${image} &> /dev/null
        if [ $? == 0 ]; then
            echo "Pod ${pod_name} updated"
            pods_restart_count="${pods_restart_count}${pod_name},${pod_restart_count} "
        else
            echo "Pod ${pod_name} update failed"
        fi
    fi
done

# change image for rs
kubectl set image rs ${rs} -n ${namespace} ${container}=${image} &> /dev/null
if [ $? == 0 ]; then
    echo "Replicaset ${rs} updated"
else
    echo "Replicaset ${rs} update failed"
    kubectl rollout resume deployment ${name} -n ${namespace}
    exit 1
fi

# change image for deployment
kubectl set image deployment ${name} -n ${namespace} ${container}=${image} &> /dev/null
if [ $? == 0 ]; then
    echo "Deployment ${name} updated"
else
    echo "Deployment ${name} update failed"
    kubectl rollout resume deployment ${name} -n ${namespace}
    exit 1
fi
# resume deployment
kubectl rollout resume deployment ${name} -n ${namespace}

echo "Deployment ${name} change to ${image} completed successfully"
echo "Waiting for pods to be ready..."
for prc in ${pods_restart_count}; do
    # wait for pods container restartCount+1
    pod_name=$(echo ${prc} | cut -d, -f1)
    old_count=$(echo ${prc} | cut -d, -f2)
    cur_count=$(kubectl get pod ${pod_name} -n ${namespace} -o jsonpath="{.status.containerStatuses[?(@.name==\"${container}\")].restartCount}")
    while [ ${cur_count} -le ${old_count} ]; do
        sleep 2
        cur_count=$(kubectl get pod ${pod_name} -n ${namespace} -o jsonpath="{.status.containerStatuses[?(@.name==\"${container}\")].restartCount}")
    done
    echo "Pod ${pod_name} is ready"
done
echo "All pods are ready"




