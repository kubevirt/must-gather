module github.com/kubevirt/must-gather/tests

go 1.16

require (
	github.com/Azure/go-autorest/autorest v0.11.18
	github.com/Azure/go-autorest/autorest/adal v0.9.13
	github.com/davecgh/go-spew v1.1.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.5
	github.com/google/gofuzz v1.1.0
	github.com/googleapis/gnostic v0.5.5
	github.com/imdario/mergo v0.3.5
	github.com/json-iterator/go v1.1.11
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/modern-go/reflect2 v1.0.1
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	golang.org/x/net v0.0.0-20210520170846-37e1c6afe023
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20210616094352-59db8d763f22
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d
	golang.org/x/text v0.3.6
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac
	google.golang.org/protobuf v1.26.0
	gopkg.in/inf.v0 v0.9.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/client-go v0.22.0
	k8s.io/klog/v2 v2.9.0
	k8s.io/utils v0.0.0-20210707171843-4b05e18ac7d9
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2
	sigs.k8s.io/yaml v1.2.0
)

exclude k8s.io/cluster-bootstrap v0.0.0

exclude k8s.io/api v0.0.0

exclude k8s.io/apiextensions-apiserver v0.0.0

exclude k8s.io/apimachinery v0.0.0

exclude k8s.io/apiserver v0.0.0

exclude k8s.io/code-generator v0.0.0

exclude k8s.io/component-base v0.0.0

exclude k8s.io/kube-aggregator v0.0.0

exclude k8s.io/cli-runtime v0.0.0

exclude k8s.io/kubectl v0.0.0

exclude k8s.io/cloud-provider v0.0.0

exclude k8s.io/cri-api v0.0.0

exclude k8s.io/csi-translation-lib v0.0.0

exclude k8s.io/kube-controller-manager v0.0.0

exclude k8s.io/kube-proxy v0.0.0

exclude k8s.io/kube-scheduler v0.0.0

exclude k8s.io/kubelet v0.0.0

exclude k8s.io/legacy-cloud-providers v0.0.0

exclude k8s.io/metrics v0.0.0

exclude k8s.io/sample-apiserver v0.0.0

replace (
	github.com/appscode/jsonpatch => github.com/appscode/jsonpatch v1.0.1
	github.com/go-kit/kit => github.com/go-kit/kit v0.3.0
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.5.1 // indirect
	github.com/openshift/machine-api-operator => github.com/openshift/machine-api-operator v0.2.1-0.20191025120018-fb3724fc7bdf
)

// Aligning with https://github.com/kubevirt/containerized-data-importer/blob/release-v1.34.1
replace (
	github.com/openshift/api => github.com/openshift/api v0.0.0-20201120165435-072a4cd8ca42
	github.com/openshift/library-go => github.com/mhenriks/library-go v0.0.0-20200804184258-4fc3a5379c7a
	sigs.k8s.io/structured-merge-diff => sigs.k8s.io/structured-merge-diff v0.0.0-20190302045857-e85c7b244fd2
)

replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm

replace vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787

replace bitbucket.org/ww/goautoneg => github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d

// Fixes various security issues forcing newer versions of affected dependencies,
// prune the list once not explicitly required
replace (
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/gorilla/websocket => github.com/gorilla/websocket v1.4.2
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/crypto/ssh/terminal => golang.org/x/crypto/ssh/terminal v0.0.0-20201221181555-eec23a3978ad
)

exclude github.com/openshift/client-go v0.0.0
