# csi-cloudscale 
A Container Storage Interface ([CSI](https://github.com/container-storage-interface/spec)) driver 
for cloudscale.ch volumes. The CSI plugin allows you to use cloudscale.ch volumes with your 
preferred Container Orchestrator.

The cloudscale.ch CSI plugin is mostly tested on kubernetes. Theoretically, it should also work with
other container orchestrators.

This fork is ahead of cloudscale.ch's CSI driver and adds the following features:

* Support for CSI spec v1.0.0 (PR [#4](https://github.com/cloudscale-ch/csi-cloudscale/pull/4) submitted)
* Support for `bulk` volumes (PR pending)
* Change ext4 formatting behaviour to not reserve any diskspace for root (PR pending)
* Support for luks encrypted volumes (PR pending)

## Releases

The cloudscale.ch CSI plugin follows [semantic versioning](https://semver.org/).
The current version of linkyard's fork is: **`1.0.0`**. 

## Volume parameters

This plugin supports the following volume parameters (in case of kubernetes: parameters on the 
`StorageClass` object):

* `csi.cloudscale.ch/volume-type`: `ssd` or `bulk`; defaults to `ssd` if not set

For LUKS encryption:

* `csi.cloudscale.ch/luks-encrypted`: set to the string `"true"` if the volume should be encrypted
  with LUKS
* `csi.cloudscale.ch/luks-cipher`: cipher to use; must be supported by the kernel and luks, we
  suggest `aes-xts-plain64`
* `csi.cloudscale.ch/luks-key-size`: key-size to use; we suggest `512` for `aes-xts-plain64`

For LUKS encrypted volumes, a secret that contains the LUKS key needs to be referenced through
the `csiNodeStageSecretName` and `csiNodeStageSecretNamespace` parameter. See the included 
`StorageClass` definitions for examples.

## Installing to Kubernetes

**Requirements:**

* Kubernetes v1.13.x
* `--allow-privileged` flag must be set to true for both the API server and the kubelet
* (if you use Docker) the Docker daemon of the cluster nodes must allow shared mounts

To install the CSI plugin on kubernetes, the following steps are necessary:

* Create a secret with your cloudscale.ch API access token
* Deploy the CSI plugin
* Use on of the pre-defined storage-classes or create you own storage classes

### Pre-defined storage classes

The default deployment bundled in the `deploy/kubernetes/releases` folder includes three storage
classes:

* `cloudscale-volume-ssd` - the default storage class; uses an ssd volume, no luks encryption
* `cloudscale-volume-bulk` - uses a bulk volume, no luks encryption
* `cloudscale-volume-ssd-luks` - uses an ssd volume that will be encrypyted with luks; a luks-key
  must be supplied
* `cloudscale-volume-bulk-luks` - uses a bulk volume that will be encrypyted with luks; a luks-key
  must be supplied

To use one of the shipped luks storage classes, you need to create a secret named 
`${pvc.name}-luks-key` in the same namespace as the persistent volume claim. The secret must
contain an element called `luksKey` that will be used as the luks encryption key.

Example: If you create a persistent volume claim with the name `my-pvc`, you need to create a
secret `my-pvc-luks-key`. 

### 1. Create a secret with your cloudscale.ch API Access Token:

Replace the placeholder string starting with `a05...` with your own secret and
save it as `secret.yml`: 

```
apiVersion: v1
kind: Secret
metadata:
  name: cloudscale
  namespace: kube-system
stringData:
  access-token: "a05dd2f26b9b9ac2asdas__REPLACE_ME____123cb5d1ec17513e06da"
```

and create the secret using kubectl:

```
$ kubectl create -f ./secret.yml
secret "cloudscale" created
```

### 2. Deploy the CSI plugin and sidecars:

Before you continue, be sure to checkout to a 
[tagged release](https://github.com/linkyard/csi-cloudscale/releases). 
Always use the [latest stable version](https://github.com/linkyard/csi-cloudscale/releases/latest) 
For example, to use the latest stable version (`1.0.0`) you can execute the following command:

```
$ kubectl apply -f https://raw.githubusercontent.com/linkyard/csi-cloudscale/master/deploy/kubernetes/releases/csi-cloudscale-linkyard-v1.0.0.yaml
```

This file will be always updated to point to the latest stable release.

There are also `dev` images available:

```
$ kubectl apply -f https://raw.githubusercontent.com/linkyard/csi-cloudscale/master/deploy/kubernetes/releases/csi-cloudscale-linkyard-dev.yaml
```

The storage classes `cloudscale-volume-ssd`, `cloudscale-volume-bulk`, `cloudscale-volume-ssd-luks`
and `cloudscale-volume-bulk-luks` will be created. The storage-class `cloudscale-volume-ssd` will
be the **"default"** for dynamic provisioning. 

#### 3. Test and verify:

Create a PersistentVolumeClaim. This makes sure a volume is created and provisioned on your behalf:

```
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: csi-pvc
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
  storageClassName: cloudscale-volume-ssd
```

Check that a new `PersistentVolume` is created based on your claim:

```
$ kubectl get pv
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS    CLAIM             STORAGECLASS            REASON    AGE
pvc-0879b207-9558-11e8-b6b4-5218f75c62b9   5Gi        RWO            Delete           Bound     default/csi-pvc   cloudscale-volume-ssd             3m
```

The above output means that the CSI plugin successfully created (provisioned) a
new Volume on behalf of you. You should be able to see this newly created
volumes in the server detail view in the cloudscale.ch UI.

The volume is not attached to any node yet. It'll only attached to a node if a
workload (i.e: pod) is scheduled to a specific node. Now let us create a Pod
that refers to the above volume. When the Pod is created, the volume will be
attached, formatted and mounted to the specified Container:

```
kind: Pod
apiVersion: v1
metadata:
  name: my-csi-app
spec:
  containers:
    - name: my-frontend
      image: busybox
      volumeMounts:
      - mountPath: "/data"
        name: my-cloudscale-volume
      command: [ "sleep", "1000000" ]
  volumes:
    - name: my-cloudscale-volume
      persistentVolumeClaim:
        claimName: csi-pvc 
```

Check if the pod is running successfully:


```
$ kubectl describe pods/my-csi-app
```

Write inside the app container:

```
$ kubectl exec -ti my-csi-app /bin/sh
/ # touch /data/hello-world
/ # exit
$ kubectl exec -ti my-csi-app /bin/sh
/ # ls /data
hello-world
```

## Development

Requirements:

* Go: min `v1.10.x`

After making your changes, run the unit tests: 

```
$ make test
```

To run the integration tests run the following:

```
$ KUBECONFIG=$(pwd)/kubeconfig make test-integration
```

## Contributing

At cloudscale.ch we value and love our community! If you have any issues or
would like to contribute, feel free to open an issue/PR
