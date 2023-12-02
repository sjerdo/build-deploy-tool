package cmd

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/uselagoon/build-deploy-tool/internal/helpers"
	"github.com/uselagoon/build-deploy-tool/internal/lagoon"
	"github.com/uselagoon/build-deploy-tool/internal/testdata"
)

func TestIdentifyRoute(t *testing.T) {
	tests := []struct {
		name         string
		args         testdata.TestData
		templatePath string
		want         string
		wantJSON     string
		wantRemain   []string
		wantautoGen  []string
	}{
		{
			name: "test1 check LAGOON_FASTLY_SERVICE_IDS with secret no values",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "main",
					Branch:          "main",
					LagoonYAML:      "../internal/testdata/node/lagoon.yml",
					ProjectVariables: []lagoon.EnvironmentVariable{
						{
							Name:  "LAGOON_FASTLY_SERVICE_IDS",
							Value: "example.com:service-id:true:annotationscom",
							Scope: "build",
						},
					},
				}, true),
			templatePath: "testdata/output",
			want:         "https://example.com",
			wantRemain:   []string{"https://node-example-project-main.example.com", "https://example.com"},
			wantautoGen:  []string{"https://node-example-project-main.example.com"},
			wantJSON:     `{"primary":"https://example.com","secondary":["https://node-example-project-main.example.com","https://example.com"],"autogenerated":["https://node-example-project-main.example.com"]}`,
		},
		{
			name: "test2 check LAGOON_FASTLY_SERVICE_IDS no secret and no values",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "main",
					Branch:          "main",
					LagoonYAML:      "../internal/testdata/node/lagoon.yml",
					ProjectVariables: []lagoon.EnvironmentVariable{
						{
							Name:  "LAGOON_FASTLY_SERVICE_IDS",
							Value: "example.com:service-id:true",
							Scope: "build",
						},
					},
				}, true),
			templatePath: "testdata/output",
			want:         "https://example.com",
			wantRemain:   []string{"https://node-example-project-main.example.com", "https://example.com"},
			wantautoGen:  []string{"https://node-example-project-main.example.com"},
			wantJSON:     `{"primary":"https://example.com","secondary":["https://node-example-project-main.example.com","https://example.com"],"autogenerated":["https://node-example-project-main.example.com"]}`,
		},
		{
			name: "test3 check LAGOON_FASTLY_SERVICE_ID no secret and no values",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "main",
					Branch:          "main",
					LagoonYAML:      "../internal/testdata/node/lagoon.yml",
					ProjectVariables: []lagoon.EnvironmentVariable{
						{
							Name:  "LAGOON_FASTLY_SERVICE_ID",
							Value: "service-id:true",
							Scope: "build",
						},
					},
				}, true),
			templatePath: "testdata/output",
			want:         "https://example.com",
			wantRemain:   []string{"https://node-example-project-main.example.com", "https://example.com"},
			wantautoGen:  []string{"https://node-example-project-main.example.com"},
			wantJSON:     `{"primary":"https://example.com","secondary":["https://node-example-project-main.example.com","https://example.com"],"autogenerated":["https://node-example-project-main.example.com"]}`,
		},
		{
			name: "test4 check no fastly and no values",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "main",
					Branch:          "main",
					LagoonYAML:      "../internal/testdata/node/lagoon.yml",
				}, true),
			templatePath: "testdata/output",
			want:         "https://example.com",
			wantRemain:   []string{"https://node-example-project-main.example.com", "https://example.com"},
			wantautoGen:  []string{"https://node-example-project-main.example.com"},
			wantJSON:     `{"primary":"https://example.com","secondary":["https://node-example-project-main.example.com","https://example.com"],"autogenerated":["https://node-example-project-main.example.com"]}`,
		},
		{
			name: "test5 multiproject1 no values",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "multiproject1",
					EnvironmentName: "multiproject",
					Branch:          "multiproject",
					LagoonYAML:      "../internal/testdata/node/lagoon.polysite.yml",
				}, true),
			templatePath: "testdata/output",
			want:         "https://multiproject1.com",
			wantRemain:   []string{"https://node-multiproject1-multiproject.example.com", "https://multiproject1.com"},
			wantautoGen:  []string{"https://node-multiproject1-multiproject.example.com"},
			wantJSON:     `{"primary":"https://multiproject1.com","secondary":["https://node-multiproject1-multiproject.example.com","https://multiproject1.com"],"autogenerated":["https://node-multiproject1-multiproject.example.com"]}`,
		},
		{
			name: "test6 multiproject2 no values",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "multiproject2",
					EnvironmentName: "multiproject",
					Branch:          "multiproject",
					LagoonYAML:      "../internal/testdata/node/lagoon.polysite.yml",
				}, true),
			templatePath: "testdata/output",
			want:         "https://multiproject2.com",
			wantRemain:   []string{"https://node-multiproject2-multiproject.example.com", "https://multiproject2.com"},
			wantautoGen:  []string{"https://node-multiproject2-multiproject.example.com"},
			wantJSON:     `{"primary":"https://multiproject2.com","secondary":["https://node-multiproject2-multiproject.example.com","https://multiproject2.com"],"autogenerated":["https://node-multiproject2-multiproject.example.com"]}`,
		},
		{
			name: "test7 multidomain no values",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "tworoutes",
					Branch:          "tworoutes",
					LagoonYAML:      "../internal/testdata/node/lagoon.yml",
				}, true),
			templatePath: "testdata/output",
			want:         "https://example.com",
			wantRemain:   []string{"https://node-example-project-tworoutes.example.com", "https://example.com", "https://www.example.com"},
			wantautoGen:  []string{"https://node-example-project-tworoutes.example.com"},
			wantJSON:     `{"primary":"https://example.com","secondary":["https://node-example-project-tworoutes.example.com","https://example.com","https://www.example.com"],"autogenerated":["https://node-example-project-tworoutes.example.com"]}`,
		},
		{
			name: "test8 multidomain no values",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "branch-routes",
					Branch:          "branch/routes",
					LagoonYAML:      "../internal/testdata/node/lagoon.yml",
				}, true),
			templatePath: "testdata/output",
			want:         "https://customdomain-will-be-main-domain.com",
			wantRemain:   []string{"https://node-example-project-branch-routes.example.com", "https://customdomain-will-be-main-domain.com", "https://customdomain-will-be-not-be-main-domain.com"},
			wantautoGen:  []string{"https://node-example-project-branch-routes.example.com"},
			wantJSON:     `{"primary":"https://customdomain-will-be-main-domain.com","secondary":["https://node-example-project-branch-routes.example.com","https://customdomain-will-be-main-domain.com","https://customdomain-will-be-not-be-main-domain.com"],"autogenerated":["https://node-example-project-branch-routes.example.com"]}`,
		},
		{
			name: "test9 active no values",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:        "example-project",
					EnvironmentName:    "main",
					Branch:             "main",
					ActiveEnvironment:  "main",
					StandbyEnvironment: "main-sb",
					LagoonYAML:         "../internal/testdata/node/lagoon.activestandby.yml",
				}, true),
			templatePath: "testdata/output",
			want:         "https://active.example.com",
			wantRemain:   []string{"https://node-example-project-main.example.com", "https://main.example.com", "https://active.example.com"},
			wantautoGen:  []string{"https://node-example-project-main.example.com"},
			wantJSON:     `{"primary":"https://active.example.com","secondary":["https://node-example-project-main.example.com","https://main.example.com","https://active.example.com"],"autogenerated":["https://node-example-project-main.example.com"]}`,
		},
		{
			name: "test10 standby no values",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:        "example-project",
					EnvironmentName:    "main-sb",
					Branch:             "main-sb",
					ActiveEnvironment:  "main",
					StandbyEnvironment: "main-sb",
					LagoonYAML:         "../internal/testdata/node/lagoon.activestandby.yml",
				}, true),
			templatePath: "testdata/output",
			want:         "https://standby.example.com",
			wantRemain:   []string{"https://node-example-project-main-sb.example.com", "https://main-sb.example.com", "https://standby.example.com"},
			wantautoGen:  []string{"https://node-example-project-main-sb.example.com"},
			wantJSON:     `{"primary":"https://standby.example.com","secondary":["https://node-example-project-main-sb.example.com","https://main-sb.example.com","https://standby.example.com"],"autogenerated":["https://node-example-project-main-sb.example.com"]}`,
		},
		{
			name: "test11 no custom ingress",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "no-ingress",
					Branch:          "no-ingress",
					LagoonYAML:      "../internal/testdata/node/lagoon.yml",
				}, true),
			templatePath: "testdata/output",
			want:         "https://node-example-project-no-ingress.example.com",
			wantRemain:   []string{"https://node-example-project-no-ingress.example.com"},
			wantautoGen:  []string{"https://node-example-project-no-ingress.example.com"},
			wantJSON:     `{"primary":"https://node-example-project-no-ingress.example.com","secondary":["https://node-example-project-no-ingress.example.com"],"autogenerated":["https://node-example-project-no-ingress.example.com"]}`,
		},
		{
			name: "test12 no custom ingress",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "no-ingress",
					Branch:          "no-ingress",
					LagoonYAML:      "../internal/testdata/node/lagoon.autogen-prefixes-1.yml",
				}, true),
			templatePath: "testdata/output",
			want:         "https://node-example-project-no-ingress.example.com",
			wantRemain: []string{
				"https://node-example-project-no-ingress.example.com",
				"https://www.node-example-project-no-ingress.example.com",
				"https://en.node-example-project-no-ingress.example.com",
				"https://de.node-example-project-no-ingress.example.com",
				"https://fi.node-example-project-no-ingress.example.com",
			},
			wantautoGen: []string{
				"https://node-example-project-no-ingress.example.com",
				"https://www.node-example-project-no-ingress.example.com",
				"https://en.node-example-project-no-ingress.example.com",
				"https://de.node-example-project-no-ingress.example.com",
				"https://fi.node-example-project-no-ingress.example.com",
			},
			wantJSON: `{"primary":"https://node-example-project-no-ingress.example.com","secondary":["https://node-example-project-no-ingress.example.com","https://www.node-example-project-no-ingress.example.com","https://en.node-example-project-no-ingress.example.com","https://de.node-example-project-no-ingress.example.com","https://fi.node-example-project-no-ingress.example.com"],"autogenerated":["https://node-example-project-no-ingress.example.com","https://www.node-example-project-no-ingress.example.com","https://en.node-example-project-no-ingress.example.com","https://de.node-example-project-no-ingress.example.com","https://fi.node-example-project-no-ingress.example.com"]}`,
		},
		{
			name: "test13 no autogenerated routes",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "main",
					Branch:          "main",
					LagoonYAML:      "../internal/testdata/node/lagoon.autogen-1.yml",
				}, true),
			templatePath: "testdata/output",
			want:         "https://example.com",
			wantRemain:   []string{"https://example.com"},
			wantautoGen:  []string{},
			wantJSON:     `{"primary":"https://example.com","secondary":["https://example.com"],"autogenerated":[]}`,
		},
		{
			name: "test14 only autogenerated route",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "notmain",
					Branch:          "notmain",
					LagoonYAML:      "../internal/testdata/node/lagoon.yml",
				}, true),
			templatePath: "testdata/output",
			want:         "https://node-example-project-notmain.example.com",
			wantRemain:   []string{"https://node-example-project-notmain.example.com"},
			wantautoGen:  []string{"https://node-example-project-notmain.example.com"},
			wantJSON:     `{"primary":"https://node-example-project-notmain.example.com","secondary":["https://node-example-project-notmain.example.com"],"autogenerated":["https://node-example-project-notmain.example.com"]}`,
		},
		{
			name: "test15 only autogenerated route complex",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "sales-customer-support",
					EnvironmentName: "develop",
					Branch:          "develop",
					EnvironmentType: "development",
					LagoonYAML:      "../internal/testdata/complex/lagoon.small.yml",
					ProjectVariables: []lagoon.EnvironmentVariable{
						{
							Name:  "LAGOON_SYSTEM_ROUTER_PATTERN",
							Value: "${service}-${project}-${environment}.ex1.example-web.com",
							Scope: "internal_system",
						},
					},
				}, false),
			templatePath: "testdata/output",
			want:         "https://nginx-sales-customer-support-develop.ex1.example-web.com",
			wantRemain:   []string{"https://nginx-sales-customer-support-develop.ex1.example-web.com"},
			wantautoGen:  []string{"https://nginx-sales-customer-support-develop.ex1.example-web.com"},
			wantJSON:     `{"primary":"https://nginx-sales-customer-support-develop.ex1.example-web.com","secondary":["https://nginx-sales-customer-support-develop.ex1.example-web.com"],"autogenerated":["https://nginx-sales-customer-support-develop.ex1.example-web.com"]}`,
		},
		{
			name: "test16 autogenerated routes where lagoon.name of service does not match service names",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "content-example-com",
					EnvironmentName: "feature-migration",
					Branch:          "feature/migration",
					EnvironmentType: "development",
					LagoonYAML:      "../internal/testdata/nginxphp/lagoon.nginx-2.yml",
					ProjectVariables: []lagoon.EnvironmentVariable{
						{
							Name:  "LAGOON_SYSTEM_ROUTER_PATTERN",
							Value: "${environment}.${project}.example.com",
							Scope: "internal_system",
						},
					},
				}, false),
			templatePath: "testdata/output",
			want:         "https://nginx-php.feature-migration.content-example-com.example.com",
			wantRemain:   []string{"https://nginx-php.feature-migration.content-example-com.example.com"},
			wantautoGen:  []string{"https://nginx-php.feature-migration.content-example-com.example.com"},
			wantJSON:     `{"primary":"https://nginx-php.feature-migration.content-example-com.example.com","secondary":["https://nginx-php.feature-migration.content-example-com.example.com"],"autogenerated":["https://nginx-php.feature-migration.content-example-com.example.com"]}`,
		},
		{
			name: "test17 autogenerated routes with mulitple routeable services",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "main",
					Branch:          "main",
					LagoonYAML:      "../internal/testdata/complex/lagoon.small-2.yml",
				}, true),
			templatePath: "testdata/output",
			want:         "https://nginx-example-project-main.example.com",
			wantRemain:   []string{"https://nginx-example-project-main.example.com", "https://varnish-example-project-main.example.com"},
			wantautoGen:  []string{"https://nginx-example-project-main.example.com", "https://varnish-example-project-main.example.com"},
			wantJSON:     `{"primary":"https://nginx-example-project-main.example.com","secondary":["https://nginx-example-project-main.example.com","https://varnish-example-project-main.example.com"],"autogenerated":["https://nginx-example-project-main.example.com","https://varnish-example-project-main.example.com"]}`,
		},
		{
			name: "test18 autogenerated routes with wildcard and altnames",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "main",
					Branch:          "main",
					LagoonYAML:      "../internal/testdata/complex/lagoon.complex-2.yml",
				}, true),
			templatePath: "testdata/output",
			want:         "https://wild.example.com",
			wantRemain:   []string{"https://nginx-example-project-main.example.com", "https://wild.example.com", "https://alt.example.com", "https://www.example.com", "https://en.example.com"},
			wantautoGen:  []string{"https://nginx-example-project-main.example.com"},
			wantJSON:     `{"primary":"https://wild.example.com","secondary":["https://nginx-example-project-main.example.com","https://wild.example.com","https://alt.example.com","https://www.example.com","https://en.example.com"],"autogenerated":["https://nginx-example-project-main.example.com"]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set the environment variables from args
			savedTemplates := tt.templatePath
			generator, err := testdata.SetupEnvironment(*rootCmd, savedTemplates, tt.args)
			if err != nil {
				t.Errorf("%v", err)
			}
			primary, remainders, autogen, err := IdentifyPrimaryIngress(generator)
			if err != nil {
				t.Errorf("%v", err)
			}

			if primary != tt.want {
				t.Errorf("returned primary %v doesn't match want %v", primary, tt.want)
			}

			if !reflect.DeepEqual(remainders, tt.wantRemain) {
				t.Errorf("returned remainders %v doesn't match want %v", remainders, tt.wantRemain)
			}

			if !reflect.DeepEqual(autogen, tt.wantautoGen) {
				t.Errorf("returned autogen %v doesn't match want %v", autogen, tt.wantautoGen)
			}

			ret := ingressIdentifyJSON{
				Primary:       primary,
				Secondary:     remainders,
				Autogenerated: autogen,
			}
			retJSON, _ := json.Marshal(ret)

			if string(retJSON) != tt.wantJSON {
				t.Errorf("returned autogen %v doesn't match want %v", string(retJSON), tt.wantJSON)
			}
			t.Cleanup(func() {
				helpers.UnsetEnvVars(nil)
			})
		})
	}
}

func TestCreatedIngressIdentification(t *testing.T) {
	type args struct {
		alertContact       string
		statusPageID       string
		projectName        string
		environmentName    string
		branch             string
		prNumber           string
		prHeadBranch       string
		prBaseBranch       string
		environmentType    string
		buildType          string
		activeEnvironment  string
		standbyEnvironment string
		cacheNoCache       string
		serviceID          string
		secretPrefix       string
		projectVars        string
		envVars            string
		lagoonVersion      string
		lagoonYAML         string
		valuesFilePath     string
		templatePath       string
	}
	tests := []struct {
		name         string
		args         testdata.TestData
		templatePath string
		want         string
		wantJSON     string
		wantRemain   []string
		wantautoGen  []string
	}{
		{
			name: "test13 no autogenerated routes",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "main",
					Branch:          "main",
					LagoonYAML:      "../internal/testdata/node/lagoon.autogen-1.yml",
				}, true),
			templatePath: "testdata/output",
			wantRemain:   []string{"example.com"},
			wantautoGen:  []string{},
			wantJSON:     `{"primary":"","secondary":["example.com"],"autogenerated":[]}`,
		},
		{
			name: "test14 only autogenerated route",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "notmain",
					Branch:          "notmain",
					LagoonYAML:      "../internal/testdata/node/lagoon.yml",
				}, true),
			templatePath: "testdata/output",
			wantRemain:   []string{},
			wantautoGen:  []string{"node"},
			wantJSON:     `{"primary":"","secondary":[],"autogenerated":["node"]}`,
		},
		{
			name: "test15 only autogenerated route complex",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "sales-customer-support",
					EnvironmentName: "develop",
					Branch:          "develop",
					EnvironmentType: "development",
					LagoonYAML:      "../internal/testdata/complex/lagoon.small.yml",
					ProjectVariables: []lagoon.EnvironmentVariable{
						{
							Name:  "LAGOON_SYSTEM_ROUTER_PATTERN",
							Value: "${service}-${project}-${environment}.ex1.example-web.com",
							Scope: "internal_system",
						},
					},
				}, false),
			templatePath: "testdata/output",
			wantRemain:   []string{},
			wantautoGen:  []string{"nginx"},
			wantJSON:     `{"primary":"","secondary":[],"autogenerated":["nginx"]}`,
		},
		{
			name: "test16 autogenerated routes where lagoon.name of service does not match service names",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "content-example-com",
					EnvironmentName: "feature-migration",
					Branch:          "feature/migration",
					EnvironmentType: "development",
					LagoonYAML:      "../internal/testdata/nginxphp/lagoon.nginx-2.yml",
					ProjectVariables: []lagoon.EnvironmentVariable{
						{
							Name:  "LAGOON_SYSTEM_ROUTER_PATTERN",
							Value: "${environment}.${project}.example.com",
							Scope: "internal_system",
						},
					},
				}, false),
			templatePath: "testdata/output",
			wantRemain:   []string{},
			wantautoGen:  []string{"nginx-php"},
			wantJSON:     `{"primary":"","secondary":[],"autogenerated":["nginx-php"]}`,
		},
		{
			name: "test17 autogenerated routes with mulitple routeable services",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "main",
					Branch:          "main",
					LagoonYAML:      "../internal/testdata/complex/lagoon.small-2.yml",
				}, true),
			templatePath: "testdata/output",
			wantRemain:   []string{},
			wantautoGen:  []string{"nginx", "varnish"},
			wantJSON:     `{"primary":"","secondary":[],"autogenerated":["nginx","varnish"]}`,
		},
		{
			name: "test18 autogenerated routes with wildcard and altnames",
			args: testdata.GetSeedData(
				testdata.TestData{
					ProjectName:     "example-project",
					EnvironmentName: "main",
					Branch:          "main",
					LagoonYAML:      "../internal/testdata/complex/lagoon.complex-2.yml",
				}, true),
			templatePath: "testdata/output",
			wantRemain:   []string{"wildcard-wild.example.com", "alt.example.com"},
			wantautoGen:  []string{"nginx"},
			wantJSON:     `{"primary":"","secondary":["wildcard-wild.example.com","alt.example.com"],"autogenerated":["nginx"]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set the environment variables from args
			savedTemplates := tt.templatePath
			generator, err := testdata.SetupEnvironment(*rootCmd, savedTemplates, tt.args)
			if err != nil {
				t.Errorf("%v", err)
			}

			autogen, remainders, err := CreatedIngressIdentification(generator)
			if err != nil {
				t.Errorf("%v", err)
			}

			if !reflect.DeepEqual(autogen, tt.wantautoGen) {
				t.Errorf("returned autogen %v doesn't match want %v", autogen, tt.wantautoGen)
			}

			if !reflect.DeepEqual(remainders, tt.wantRemain) {
				t.Errorf("returned remainders %v doesn't match want %v", remainders, tt.wantRemain)
			}

			ret := ingressIdentifyJSON{
				Autogenerated: autogen,
				Secondary:     remainders,
			}
			retJSON, _ := json.Marshal(ret)

			if string(retJSON) != tt.wantJSON {
				t.Errorf("returned autogen %v doesn't match want %v", string(retJSON), tt.wantJSON)
			}
		})
	}
}
