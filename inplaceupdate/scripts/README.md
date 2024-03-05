## 使用方法
脚本接收4个参数：
- Deployment名称
- Deployment的namespace
- Deployment的container名称
- Deployment的container的镜像

4个参数缺一不可， 且顺序不能错。

```shell
scripts|main⚡ ⇒ bash inplaceupdate.sh help                            
Usage: inplaceupdate.sh <name> <namespace> <container> <image>
```
脚本执行后，会修改pod、rs、deployment的镜像， 但不会删除pod， pod的属性也不会变更。

检查原地升级是否成功的方法为查看
- pod的镜像是否变更
- pod restart次数+1

## 执行
```shell
scripts|main⚡ ⇒ bash  inplaceupdate.sh nginx default nginx nginx:1.25  
deployment.apps/nginx paused
Pod nginx-54b596f5bf-qwgkl updated
Replicaset nginx-54b596f5bf updated
Deployment nginx updated
deployment.apps/nginx resumed
Deployment nginx change to nginx:1.25 completed successfully
Waiting for pods to be ready...
Pod nginx-54b596f5bf-qwgkl is ready
All pods are ready
```