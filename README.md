# sample-gameserver-lister

This sample program shows Agones gameserver's name and state and IPs(external/internal) from K8s Lister

## Usage

1. Start K8s cluster
```bash
# for example
minikube start --driver hyperkit
``` 

2. Install Agones
```bash
# for example
kubectl create namespace agones-system
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/release-1.21.0/install/yaml/install.yaml
```

3. Run this program
```bash
go run main.go
```
