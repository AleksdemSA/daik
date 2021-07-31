package main

import (
	"color"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func stage(message string) {
	fmt.Println()
	fmt.Println(color.Blue + "##### " + message + " #####" + color.Reset)
	fmt.Println()

}

func stageSuccess(message string) {
	fmt.Println(color.Green + message + color.Reset)
}

func stageError(message string) {
	fmt.Println()
	fmt.Println(color.Red + message + color.Reset)
	fmt.Println()
}

func debug(message string) {
	fmt.Println()
	fmt.Println(color.Yellow + message + color.Reset)
	fmt.Println()
}

func applyConfig(message string) {

	f, err := os.Create("tmp.yml")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err2 := f.WriteString(message)
	if err2 != nil {
		log.Fatal(err2)
	}

	out, err := exec.Command("kubectl", "apply", "-f", "tmp.yml").Output()
	if err != nil {
		stageError(err.Error())
		log.Fatal()
	}

	err = os.Remove("tmp.yml")
	if err != nil {
		log.Fatal(err)
	}

	stageSuccess(string(out))
}

func createConfig(data, username, namespace string) {

	f, err := os.Create(username + "-" + namespace + ".yml")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err2 := f.WriteString(data)
	if err2 != nil {
		log.Fatal(err2)
	}

	stageSuccess("File " + username + "-" + namespace + ".yml created")
	stageSuccess("You can run (example) kubectl --kubeconfig=" + username + "-" + namespace + ".yml get pods\n")

}

func tokenName(username, namespace string) string {

	cmd := "kubectl describe sa " + username + " -n " + namespace + " | grep \"Tokens\" | cut -d\":\" -f2 | tr -d \" \""
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		stageError(err.Error())
		log.Fatal()
	}
	outToken := string(out)
	outToken = strings.TrimSuffix(outToken, "\n")
	return outToken
}

func getToken(userTokenName, namespace string) string {

	cmd := "kubectl get secret " + userTokenName + " -n " + namespace + " -o \"jsonpath={.data.token}\" | base64 -d"
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		stageError(err.Error())
		log.Fatal()
	}

	return (string(out))
}

func getUserCert(userTokenName, namespace string) string {

	cmd := "kubectl get secret " + userTokenName + " -n " + namespace + " -o \"jsonpath={.data['ca\\.crt']}\""
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		stageError(err.Error())
		log.Fatal()
	}

	return (string(out))
}

func getServerAddress() string {

	cmd := "kubectl config view | grep server |cut -d \":\" -f2-4"
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		stageError(err.Error())
		log.Fatal()
	}
	outToken := string(out)
	outToken = strings.TrimSuffix(outToken, "\n")
	return outToken
}

type error interface {
	Error() string
}

func main() {
	//start timer
	start := time.Now()

	var namespace string
	var username string

	if len(os.Args) != 3 {
		stage("Developer Access In Kubernetes")
		stageSuccess("Tool for create config for users in Kubernetes \nWrite, please, namespace and username. Example daik devstand user")
		stageSuccess("Example to use config: kubectl --kubeconfig=CONFIGNAME.yml get pods")
		os.Exit(0)
	} else {
		namespace = os.Args[1]
		username = os.Args[2]
		stage("Parameters:")
		stageSuccess("* namespace: " + namespace)
		stageSuccess("* username: " + username)
	}

	stage("Get nodes (check connect)...")

	out, err := exec.Command("kubectl", "get", "nodes").Output()
	if err != nil {
		stageError(err.Error())
		stageError(string(out))
		log.Fatal()
	}
	stageSuccess(string(out))



	// create ns
	stage("Create NS if not exist")

	namespaceConfig := `apiVersion: v1
kind: Namespace
metadata:
  name: ` + namespace

	applyConfig(namespaceConfig)



	// create sa
	stage("Create ServiceAccount")

	serviceAccountConfig := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: ` + username + `
  namespace: ` + namespace

	applyConfig(serviceAccountConfig)



	// create role
	stage("Create Role")

	serviceRole := `kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ` + username + `-access
  namespace: ` + namespace + `
rules:
- apiGroups: ["", "extensions", "apps"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["batch"]
  resources:
  - jobs
  - cronjobs
  verbs: ["*"]`

	applyConfig(serviceRole)



	// create RoleBinding
	stage("Create RoleBinding")

	serviceRoleBinding := `
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ` + username + `-view
  namespace: ` + namespace + `
subjects:
- kind: ServiceAccount
  name: ` + username + `
  namespace: ` + namespace + `
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: ` + username + `-access
`

	applyConfig(serviceRoleBinding)


	// get token ans cert
	stage("Get token and cert")

	userTokenName := tokenName(username, namespace)
	userToken := getToken(userTokenName, namespace)
	userCert := getUserCert(userTokenName, namespace)
	serverAddress := getServerAddress()

	stageSuccess("stage SUCCESS")

	// create config
	stage("Create developer config")

	developerConfig := `apiVersion: v1
kind: Config
preferences: {}
clusters:
- cluster:
    certificate-authority-data: ` + userCert + `
    server: ` + serverAddress + `
  name: cluster
users:
- name: ` + username + `
  user:
    as-user-extra: {}
    client-key-data: ` + userCert + `
    token: ` + userToken + `
contexts:
- context:
    cluster: cluster
    namespace: ` + namespace + `
    user: ` + username + `
  name: ` + namespace + `
current-context: ` + namespace + `
`

	createConfig(developerConfig, username, namespace)

	// timer
	elapsed := time.Since(start)
	log.Printf("Execution time %s", elapsed)
}
