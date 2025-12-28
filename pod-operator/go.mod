module github.com/onexstack/kubernetes-examples/pod-operator

go 1.22

require (
	k8s.io/api v0.30.1
	k8s.io/apimachinery v0.30.1
	k8s.io/client-go v0.30.1
	sigs.k8s.io/controller-runtime v0.19.1
)

require (
	github.com/go-logr/logr v1.4.2
	github.com/gogo/protobuf v1.3.2
	github.com/google/gnostic-models v0.6.9-0.20230802173545-233f14b1be6e
	github.com/google/go-cmp v0.6.0
	github.com/google/gofuzz v1.2.0
	github.com/imdario/mergo v0.3.16
	github.com/josharian/intern v1.0.0
	github.com/mailru/easyjson v0.7.7
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/modern-go/reflect2 v1.0.2
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822
	github.com/spf13/pflag v1.0.5
	golang.org/x/net v0.25.0
	golang.org/x/oauth2 v0.21.0
	golang.org/x/sys v0.20.0
	golang.org/x/term v0.20.0
	golang.org/x/text v0.15.0
	golang.org/x/time v0.5.0
	google.golang.org/appengine v1.6.8
	google.golang.org/protobuf v1.33.0
	gopkg.in/inf.v0 v0.9.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/klog/v2 v2.120.1
	k8s.io/kube-openapi v0.20.2
	k8s.io/utils v0.0.0-20240303020801-9d7a00b7a2f8
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd
	sigs.k8s.io/structured-merge-diff/v4 v4.4.2
	sigs.k8s.io/yaml v1.4.0
)
