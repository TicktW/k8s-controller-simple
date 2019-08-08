module github.com/newgoo/k8s-controller-simple

go 1.12

require (
	github.com/astaxie/beego v1.12.0 // indirect
	github.com/shiena/ansicolor v0.0.0-20151119151921-a422bbe96644 // indirect
	k8s.io/api v0.0.0-20190731142925-739c7f7721ed
	k8s.io/apimachinery v0.0.0-20190731142807-035e418f1ad9
	k8s.io/client-go v0.0.0-20190731143132-de47f833b8db
	k8s.io/code-generator v1.2.3 // indirect
	k8s.io/klog v0.3.1
	k8s.io/sample-controller v0.0.0-20190731144349-6f8905ae4ee5 // indirect
)

replace (
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20181025213731-e84da0312774
	golang.org/x/net => golang.org/x/net v0.0.0-20190206173232-65e2d4e15006
	golang.org/x/sync => golang.org/x/sync v0.0.0-20181108010431-42b317875d0f
	golang.org/x/sys => golang.org/x/sys v0.0.0-20190209173611-3b5209105503
	golang.org/x/text => golang.org/x/text v0.3.1-0.20181227161524-e6919f6577db
	golang.org/x/tools => golang.org/x/tools v0.0.0-20190313210603-aa82965741a9
	k8s.io/api => k8s.io/api v0.0.0-20190731142925-739c7f7721ed
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190731142807-035e418f1ad9
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190731143132-de47f833b8db
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190731142644-68819053f471
)
