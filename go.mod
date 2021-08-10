module github.com/yugabyte/yb-tools

go 1.16

require (
	github.com/alexeyco/simpletable v1.0.0
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/blang/vfs v1.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v1.1.0
	github.com/go-logr/zapr v1.1.0
	github.com/go-openapi/errors v0.20.0
	github.com/go-openapi/runtime v0.19.31
	github.com/go-openapi/strfmt v0.20.1
	github.com/go-openapi/swag v0.19.15
	github.com/go-openapi/validate v0.20.2
	github.com/go-swagger/go-swagger v0.27.0 // indirect
	github.com/gocql/gocql v0.0.0-20210817081954-bc256bbb90de
	github.com/golangci/golangci-lint v1.42.1
	github.com/google/uuid v1.3.0
	github.com/icza/gox v0.0.0-20210726201659-cd40a3f8d324
	github.com/juju/go4 v0.0.0-20160222163258-40d72ab9641a // indirect
	github.com/juju/persistent-cookiejar v0.0.0-20171026135701-d5e5a8405ef9
	github.com/kr/pretty v0.2.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	github.com/spyzhov/ajson v0.4.2
	github.com/yugabyte/gocql v0.0.0-20200602185649-ef3952a45ff4
	go.uber.org/zap v1.19.0
	golang.org/x/net v0.0.0-20210825183410-e898025ed96a
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b
	google.golang.org/protobuf v1.26.0
	gopkg.in/errgo.v1 v1.0.1 // indirect
	gopkg.in/retry.v1 v1.0.3 // indirect
)

replace github.com/juju/persistent-cookiejar v0.0.0-20171026135701-d5e5a8405ef9 => github.com/andrewstuart/persistent-cookiejar v0.0.0-20181121031108-afb54bd74b6b
