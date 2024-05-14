package generator

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/drone/envsubst"
	"github.com/uselagoon/build-deploy-tool/internal/helpers"
	"github.com/uselagoon/build-deploy-tool/internal/lagoon"
	"github.com/uselagoon/build-deploy-tool/internal/servicetypes"
)

// this is a map that maps old service types to their new service types
var oldServiceMap = map[string]string{
	"mariadb-shared":        "mariadb-dbaas",
	"postgres-shared":       "postgres-dbaas",
	"mongo-shared":          "mongodb-dbaas",
	"python-ckandatapusher": "python",
	"mongo":                 "mongodb",
}

// these are lagoon types that support autogenerated routes
var supportedAutogeneratedTypes = []string{
	// "kibana", //@TODO: don't even need this anymore?
	"basic",
	"basic-persistent",
	"node",
	"node-persistent",
	"nginx",
	"nginx-php",
	"nginx-php-persistent",
	"varnish",
	"varnish-persistent",
	"python-persistent",
	"python",
}

// these service types don't have images
var ignoredImageTypes = []string{
	"mariadb-dbaas",
	"postgres-dbaas",
	"mongodb-dbaas",
}

// these are lagoon types that support autogenerated routes
var supportedDBTypes = []string{
	"mariadb",
	"mariadb-dbaas",
	"postgres",
	"postgres-dbaas",
	"mongodb",
	"mongodb-dbaas",
}

// these are lagoon types that come with resources requiring backups
var typesWithBackups = []string{
	"basic-persistent",
	"node-persistent",
	"nginx-php-persistent",
	"python-persistent",
	"varnish-persistent",
	"redis-persistent",
	"solr",
	"elasticsearch",
	"opensearch",
	"rabbitmq",
	"mongodb-dbaas",
	"mariadb-dbaas",
	"postgres-dbaas",
	"mariadb-single",
	"postgres-single",
	"mongodb-single",
}

// just some default values for services
var defaultServiceValues = map[string]map[string]string{
	"elasticsearch": {
		"persistentPath": "/usr/share/elasticsearch/data",
		"persistentSize": "5Gi",
	},
	"opensearch": {
		"persistentPath": "/usr/share/opensearch/data",
		"persistentSize": "5Gi",
	},
	"mariadb-single": {
		"persistentPath": "/var/lib/mysql",
		"persistentSize": "5Gi",
	},
	"postgres-single": {
		"persistentPath": "/var/lib/postgresql/data",
		"persistentSize": "5Gi",
	},
	"mongodb-single": {
		"persistentPath": "/data/db",
		"persistentSize": "5Gi",
	},
	"varnish-persistent": {
		"persistentPath": "/var/cache/varnish",
		"persistentSize": "5Gi",
	},
	"rabbitmq": {
		"persistentPath": "/var/lib/rabbitmq",
		"persistentSize": "5Gi",
	},
	"redis-persistent": {
		"persistentPath": "/data",
		"persistentSize": "5Gi",
	},
}

// generateServicesFromDockerCompose unmarshals the docker-compose file and processes the services using composeToServiceValues
func generateServicesFromDockerCompose(
	buildValues *BuildValues,
	ignoreNonStringKeyErrors, ignoreMissingEnvFiles, debug bool,
) error {
	// take lagoon envvars and create new map for being unmarshalled against the docker-compose file
	composeVars := make(map[string]string)
	for _, envvar := range buildValues.EnvironmentVariables {
		composeVars[envvar.Name] = envvar.Value
	}

	// create the services map
	buildValues.Services = []ServiceValues{}

	// unmarshal the docker-compose.yml file
	lCompose, lComposeOrder, err := lagoon.UnmarshaDockerComposeYAML(buildValues.LagoonYAML.DockerComposeYAML, ignoreNonStringKeyErrors, ignoreMissingEnvFiles, composeVars)
	if err != nil {
		return err
	}

	// convert docker-compose services to servicevalues,
	// range over the original order of the docker-compose file when setting services
	for _, service := range lComposeOrder {
		for _, composeServiceValues := range lCompose.Services {
			if service.Name == composeServiceValues.Name {
				cService, err := composeToServiceValues(buildValues, composeServiceValues.Name, composeServiceValues, debug)
				if err != nil {
					return err
				}
				if cService.BackupsEnabled {
					buildValues.BackupsEnabled = true
				}
				buildValues.Services = append(buildValues.Services, cService)
			}
		}
	}
	return nil
}

// composeToServiceValues is the primary function used to pre-seed how templates are created
// it reads the docker-compose file and converts each service into a ServiceValues struct
// this is the "known state" of that service, and all subsequent steps to create templates will use this data unmodified
func composeToServiceValues(
	buildValues *BuildValues,
	composeService string,
	composeServiceValues composetypes.ServiceConfig,
	debug bool,
) (ServiceValues, error) {
	lagoonType := ""
	// if there are no labels, then this is probably not going to end up in Lagoon
	// the lagoonType check will skip to the end and return an empty service definition
	if composeServiceValues.Labels != nil {
		lagoonType = lagoon.CheckServiceLagoonLabel(composeServiceValues.Labels, "lagoon.type")
	}
	if lagoonType == "" {
		return ServiceValues{}, fmt.Errorf("No lagoon.type has been set for service %s. If a Lagoon service is not required, please set the lagoon.type to 'none' for this service in docker-compose.yaml. See the Lagoon documentation for supported service types.", composeService)
	} else {
		// if the lagoontype is populated, even none is valid as there may be a servicetype override in an environment variable
		autogenEnabled := true
		autogenTLSAcmeEnabled := true
		// check if autogenerated routes are disabled
		if buildValues.LagoonYAML.Routes.Autogenerate.Enabled != nil {
			if *buildValues.LagoonYAML.Routes.Autogenerate.Enabled == false {
				autogenEnabled = false
			}
		}
		// check if pullrequests autogenerated routes are disabled
		if buildValues.BuildType == "pullrequest" && buildValues.LagoonYAML.Routes.Autogenerate.AllowPullRequests != nil {
			if *buildValues.LagoonYAML.Routes.Autogenerate.AllowPullRequests == false {
				autogenEnabled = false
			} else {
				autogenEnabled = true
			}
		}
		// check if this environment has autogenerated routes disabled
		if buildValues.LagoonYAML.Environments[buildValues.Branch].AutogenerateRoutes != nil {
			if *buildValues.LagoonYAML.Environments[buildValues.Branch].AutogenerateRoutes == false {
				autogenEnabled = false
			} else {
				autogenEnabled = true
			}
		}
		// check if autogenerated routes tls-acme disabled
		if buildValues.LagoonYAML.Routes.Autogenerate.TLSAcme != nil {
			if *buildValues.LagoonYAML.Routes.Autogenerate.TLSAcme == false {
				autogenTLSAcmeEnabled = false
			}
		}
		// check lagoon yaml for an override for this service
		if value, ok := buildValues.LagoonYAML.Environments[buildValues.Environment].Types[composeService]; ok {
			lagoonType = value
		}
		// check if the service has a specific override
		serviceAutogenerated := lagoon.CheckServiceLagoonLabel(composeServiceValues.Labels, "lagoon.autogeneratedroute")
		if serviceAutogenerated != "" {
			if reflect.TypeOf(serviceAutogenerated).Kind() == reflect.String {
				vBool, err := strconv.ParseBool(serviceAutogenerated)
				if err == nil {
					autogenEnabled = vBool
				}
			}
		}
		// check if the service has a tls-acme specific override
		serviceAutogeneratedTLSAcme := lagoon.CheckServiceLagoonLabel(composeServiceValues.Labels, "lagoon.autogeneratedroute.tls-acme")
		if serviceAutogeneratedTLSAcme != "" {
			if reflect.TypeOf(serviceAutogeneratedTLSAcme).Kind() == reflect.String {
				vBool, err := strconv.ParseBool(serviceAutogeneratedTLSAcme)
				if err == nil {
					autogenTLSAcmeEnabled = vBool
				}
			}
		}
		// check if the service has a deployment servicetype override
		serviceDeploymentServiceType := lagoon.CheckServiceLagoonLabel(composeServiceValues.Labels, "lagoon.deployment.servicetype")
		if serviceDeploymentServiceType == "" {
			serviceDeploymentServiceType = composeService
		}

		// if there is a `lagoon.name` label on this service, this should be used as an override name
		lagoonOverrideName := lagoon.CheckServiceLagoonLabel(composeServiceValues.Labels, "lagoon.name")
		if lagoonOverrideName != "" {
			// if there is an override name, check all other services already existing
			for _, service := range buildValues.Services {
				// if there is an existing service with this same override name, then disable autogenerated routes
				// for this service
				if service.OverrideName == lagoonOverrideName {
					autogenEnabled = false
				}
			}
		} else {
			// otherwise just set the override name to be the service name
			lagoonOverrideName = composeService
		}

		// check if the service has any persistent labels, this is the path that the volume will be mounted to
		servicePersistentPath := lagoon.CheckServiceLagoonLabel(composeServiceValues.Labels, "lagoon.persistent")
		if servicePersistentPath == "" {
			// if there is no persistent path, check if the service type has a default path
			if val, ok := servicetypes.ServiceTypes[lagoonType]; ok {
				servicePersistentPath = val.Volumes.PersistentVolumePath
			}
		}
		servicePersistentName := lagoon.CheckServiceLagoonLabel(composeServiceValues.Labels, "lagoon.persistent.name")
		if servicePersistentName == "" && servicePersistentPath != "" {
			// if there is a persistent path defined, then set the persistent name to be the compose service if no persistent name is provided
			// persistent name is used by joined services like nginx/php or cli or worker pods to mount another service volume
			servicePersistentName = lagoonOverrideName
		}
		servicePersistentSize := lagoon.CheckServiceLagoonLabel(composeServiceValues.Labels, "lagoon.persistent.size")
		if servicePersistentSize == "" {
			// if there is no persistent size, check if the service type has a default size allocated
			if val, ok := servicetypes.ServiceTypes[lagoonType]; ok {
				servicePersistentSize = val.Volumes.PersistentVolumeSize
			}
		}

		// if there are overrides defined in the lagoon API `LAGOON_SERVICE_TYPES`
		// handle those here
		if buildValues.ServiceTypeOverrides != nil {
			serviceTypesSplit := strings.Split(buildValues.ServiceTypeOverrides.Value, ",")
			for _, sType := range serviceTypesSplit {
				sTypeSplit := strings.Split(sType, ":")
				if sTypeSplit[0] == lagoonOverrideName {
					lagoonType = sTypeSplit[1]
				}
			}
		}

		// convert old service types to new service types from the old service map
		// this allows for adding additional values to the oldServiceMap that we can force to be anything else
		if val, ok := oldServiceMap[lagoonType]; ok {
			lagoonType = val
		}

		// if there are no overrides, and the type is none, then abort here, no need to proceed calculating the type
		if lagoonType == "none" {
			return ServiceValues{}, nil
		}
		// anything after this point is where heavy processing is done as the service type has now been determined by this stage

		// handle dbaas operator checks here
		dbaasEnvironment := buildValues.EnvironmentType
		svcIsDBaaS := false
		if helpers.Contains(supportedDBTypes, lagoonType) {
			// strip the dbaas off the supplied type for checking against providers, it gets added again later
			lagoonType = strings.Split(lagoonType, "-dbaas")[0]
			err := buildValues.DBaaSClient.CheckHealth(buildValues.DBaaSOperatorEndpoint)
			if err != nil {
				// @TODO eventually this error should be handled and fail a build, with a flag to override https://github.com/uselagoon/build-deploy-tool/issues/56
				// if !buildValues.DBaaSFallbackSingle {
				// 	return ServiceValues{}, fmt.Errorf("Unable to check the DBaaS endpoint %s: %v", buildValues.DBaaSOperatorEndpoint, err)
				// }
				if debug {
					fmt.Println(fmt.Sprintf("Unable to check the DBaaS endpoint %s, falling back to %s-single: %v", buildValues.DBaaSOperatorEndpoint, lagoonType, err))
				}
				// normally we would fall back to doing a cluster capability check, this is phased out in the build tool, it isn't reliable
				// and noone should be doing checks that way any more
				// the old bash check is the following
				// elif [[ "${CAPABILITIES[@]}" =~ "mariadb.amazee.io/v1/MariaDBConsumer" ]] && ! checkDBaaSHealth ; then
				lagoonType = fmt.Sprintf("%s-single", lagoonType)
			} else {
				// if there is a `lagoon.%s-dbaas.environment` label on this service, this should be used as an the environment type for the dbaas
				dbaasLabelOverride := lagoon.CheckServiceLagoonLabel(composeServiceValues.Labels, fmt.Sprintf("lagoon.%s-dbaas.environment", lagoonType))
				if dbaasLabelOverride != "" {
					dbaasEnvironment = dbaasLabelOverride
				}

				// @TODO: maybe phase this out?
				// if value, ok := buildValues.LagoonYAML.Environments[buildValues.Environment].Overrides[composeService][mariadb][mariadb-dbaas].Environment; ok {
				// this isn't documented in the lagoon.yml, and it looks like a failover from days past.
				// 	lagoonType = value
				// }

				// if there are overrides defined in the lagoon API `LAGOON_DBAAS_ENVIRONMENT_TYPES`
				// handle those here
				exists, err := getDBaasEnvironment(buildValues, &dbaasEnvironment, lagoonOverrideName, lagoonType, debug)
				if err != nil {
					// @TODO eventually this error should be handled and fail a build, with a flag to override https://github.com/uselagoon/build-deploy-tool/issues/56
					// if !buildValues.DBaaSFallbackSingle {
					// 	return ServiceValues{}, err
					// }
					if debug {
						fmt.Println(fmt.Sprintf(
							"There was an error checking DBaaS endpoint %s, falling back to %s-single: %v",
							buildValues.DBaaSOperatorEndpoint, lagoonType, err,
						))
					}
				}

				// if the requested dbaas environment exists, then set the type to be the requested type with `-dbaas`
				if exists {
					lagoonType = fmt.Sprintf("%s-dbaas", lagoonType)
					svcIsDBaaS = true
				} else {
					// otherwise fallback to -single (if DBaaSFallbackSingle is enabled, otherwise it will error out prior)
					lagoonType = fmt.Sprintf("%s-single", lagoonType)
				}
			}
		}

		// start spot instance handling
		useSpot := false
		forceSpot := false
		cronjobUseSpot := false
		cronjobForceSpot := false
		spotTypes := ""
		cronjobSpotTypes := ""
		spotReplicas := int32(0)

		// these services can support multiple replicas in production
		// @TODO this should probably be an admin only feature flag though
		spotReplicaTypes := "nginx,nginx-persistent,nginx-php,nginx-php-persistent"
		// spotReplicaTypes := CheckAdminFeatureFlag("SPOT_REPLICAS_PRODUCTION", buildValues.EnvironmentVariables, debug) // doesn't exist yet

		productionSpot := CheckFeatureFlag("SPOT_INSTANCE_PRODUCTION", buildValues.EnvironmentVariables, debug)
		developmentSpot := CheckFeatureFlag("SPOT_INSTANCE_DEVELOPMENT", buildValues.EnvironmentVariables, debug)
		if productionSpot == "enabled" && buildValues.EnvironmentType == "production" {
			spotTypes = CheckFeatureFlag("SPOT_INSTANCE_PRODUCTION_TYPES", buildValues.EnvironmentVariables, debug)
			cronjobSpotTypes = CheckFeatureFlag("SPOT_INSTANCE_PRODUCTION_CRONJOB_TYPES", buildValues.EnvironmentVariables, debug)
		}
		if developmentSpot == "enabled" && buildValues.EnvironmentType == "development" {
			spotTypes = CheckFeatureFlag("SPOT_INSTANCE_DEVELOPMENT_TYPES", buildValues.EnvironmentVariables, debug)
			cronjobSpotTypes = CheckFeatureFlag("SPOT_INSTANCE_DEVELOPMENT_CRONJOB_TYPES", buildValues.EnvironmentVariables, debug)
		}
		// check if the provided spot instance types against the current lagoonType
		for _, t := range strings.Split(spotTypes, ",") {
			if t != "" {
				tt := strings.Split(t, ":")
				if tt[0] == lagoonType {
					useSpot = true
					if tt[1] == "force" {
						forceSpot = true
					}
				}
			}
		}
		// check if the provided cronjob spot instance types against the current lagoonType
		for _, t := range strings.Split(cronjobSpotTypes, ",") {
			if t != "" {
				tt := strings.Split(t, ":")
				if tt[0] == lagoonType {
					cronjobUseSpot = true
					if tt[1] == "force" {
						cronjobForceSpot = true
					}
				}
			}
		}
		// check if the this service is production and can support 2 replicas on spot
		for _, t := range strings.Split(spotReplicaTypes, ",") {
			if t != "" {
				if t == lagoonType && buildValues.EnvironmentType == "production" {
					spotReplicas = 2
				}
			}
		}
		// end spot instance handling

		// work out cronjobs for this service
		inpodcronjobs := []lagoon.Cronjob{}
		nativecronjobs := []lagoon.Cronjob{}
		// check if there are any duplicate named cronjobs
		if err := checkDuplicateCronjobs(buildValues.LagoonYAML.Environments[buildValues.Branch].Cronjobs); err != nil {
			return ServiceValues{}, err
		}
		if !buildValues.CronjobsDisabled {
			for _, cronjob := range buildValues.LagoonYAML.Environments[buildValues.Branch].Cronjobs {
				// if this cronjob is meant for this service, add it
				if cronjob.Service == composeService {
					var err error
					inpod, err := helpers.IsInPodCronjob(cronjob.Schedule)
					if err != nil {
						return ServiceValues{}, fmt.Errorf("Unable to validate crontab for cronjob %s: %v", cronjob.Name, err)
					}
					cronjob.Schedule, err = helpers.ConvertCrontab(buildValues.Namespace, cronjob.Schedule)
					if err != nil {
						return ServiceValues{}, fmt.Errorf("Unable to convert crontab for cronjob %s: %v", cronjob.Name, err)
					}
					if inpod {
						inpodcronjobs = append(inpodcronjobs, cronjob)
					} else {
						// make the cronjob name kubernetes compliant
						cronjob.Name = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(fmt.Sprintf("cronjob-%s-%s", lagoonOverrideName, strings.ToLower(cronjob.Name)), "-")
						if len(cronjob.Name) > 52 {
							// if the cronjob name is longer than 52 characters
							// truncate it and add a hash of the name to it
							cronjob.Name = fmt.Sprintf("%s-%s", cronjob.Name[:45], helpers.GetBase32EncodedLowercase(helpers.GetSha256Hash(cronjob.Name))[:6])
						}
						nativecronjobs = append(nativecronjobs, cronjob)
					}
				}
			}
		}

		// check if this service is one that supports autogenerated routes
		if !helpers.Contains(supportedAutogeneratedTypes, lagoonType) {
			autogenEnabled = false
			autogenTLSAcmeEnabled = false
		}

		// check if this service is one that supports backups
		backupsEnabled := false
		if helpers.Contains(typesWithBackups, lagoonType) {
			backupsEnabled = true

		}

		// create the service values
		cService := ServiceValues{
			Name:                       composeService,
			OverrideName:               lagoonOverrideName,
			Type:                       lagoonType,
			AutogeneratedRoutesEnabled: autogenEnabled,
			AutogeneratedRoutesTLSAcme: autogenTLSAcmeEnabled,
			DBaaSEnvironment:           dbaasEnvironment,
			PersistentVolumePath:       servicePersistentPath,
			PersistentVolumeName:       servicePersistentName,
			PersistentVolumeSize:       servicePersistentSize,
			UseSpotInstances:           useSpot,
			ForceSpotInstances:         forceSpot,
			CronjobUseSpotInstances:    cronjobUseSpot,
			CronjobForceSpotInstances:  cronjobForceSpot,
			Replicas:                   spotReplicas,
			InPodCronjobs:              inpodcronjobs,
			NativeCronjobs:             nativecronjobs,
			PodSecurityContext:         buildValues.PodSecurityContext,
			IsDBaaS:                    svcIsDBaaS,
			BackupsEnabled:             backupsEnabled,
		}

		// work out the images here and the associated dockerfile and contexts
		// if the type is in the ignored image types, then there is no image to build or pull for this service (eg, its a dbaas service)
		if !helpers.Contains(ignoredImageTypes, lagoonType) {
			// create a holder for all the docker related information, if this is a pull through image or a build image
			imageBuild := ImageBuild{}
			// if this is not a promote environment, then attempt to work out the image build information that is required for the builder
			if buildValues.BuildType != "promote" {
				// handle extracting the built image name from the provided image references
				if composeServiceValues.Build != nil {
					// if a build spec is defined, consume it
					// set the dockerfile
					imageBuild.DockerFile = composeServiceValues.Build.Dockerfile
					// set the context if found, otherwise set '.'
					imageBuild.Context = func(s string) string {
						if s == "" {
							return "."
						}
						return s
					}(composeServiceValues.Build.Context)
					// if there is a build target defined, set that here too
					imageBuild.Target = composeServiceValues.Build.Target
				}
				// if there is a dockerfile defined in the
				if buildValues.LagoonYAML.Environments[buildValues.Environment].Overrides[composeService].Build.Dockerfile != "" {
					imageBuild.DockerFile = buildValues.LagoonYAML.Environments[buildValues.Environment].Overrides[composeService].Build.Dockerfile
					if imageBuild.Context == "" {
						// if we get here, it means that a dockerfile override was defined in the .lagoon.yml file
						// but there was no `build` spec defined in the docker-compose file, so this just sets the context to the default `.`
						// in the same way the legacy script used to do it
						imageBuild.Context = "."
					}
				}
				if imageBuild.DockerFile == "" {
					// no dockerfile determined, this must be a pull through image
					if composeServiceValues.Image == "" {
						return ServiceValues{}, fmt.Errorf(
							"defined Dockerfile or Image for service %s defined", composeService,
						)
					}
					// check docker-compose override image
					pullImage := lagoon.CheckServiceLagoonLabel(composeServiceValues.Labels, "lagoon.image")
					// check lagoon.yml override image
					if buildValues.LagoonYAML.Environments[buildValues.Environment].Overrides[composeService].Image != "" {
						pullImage = buildValues.LagoonYAML.Environments[buildValues.Environment].Overrides[composeService].Image
					}
					if pullImage != "" {
						// if an override image is provided, envsubst it
						// not really sure why we do this, but legacy bash says `expand environment variables from ${OVERRIDE_IMAGE}`
						// so there may be some undocumented functionality that allows people to use envvars in their image overrides?
						evalImage, err := envsubst.EvalEnv(pullImage)
						if err != nil {
							return ServiceValues{}, fmt.Errorf(
								"error evaluating override image %s with envsubst", pullImage,
							)
						}
						// set the evalled image now
						pullImage = evalImage
					} else {
						// else set the pullimage to whatever is defined in the docker-compose file otherwise
						pullImage = composeServiceValues.Image
					}
					// if the image just is an image name (like "alpine") we prefix it with `libary/` as the imagecache does not understand
					// the magic `alpine` image
					if !strings.Contains(pullImage, "/") {
						imageBuild.PullImage = fmt.Sprintf("library/%s", pullImage)
					} else {
						imageBuild.PullImage = pullImage
					}
					if !ContainsRegistry(buildValues.ContainerRegistry, pullImage) {
						// if the image isn't in dockerhub, then the imagecache can't be used
						if buildValues.ImageCache != "" && strings.Count(pullImage, "/") == 1 {
							imageBuild.PullImage = fmt.Sprintf("%s%s", buildValues.ImageCache, imageBuild.PullImage)
						}
					}
				} else {
					// otherwise this must be an image build
					// set temporary image to prevent clashes?? not sure this is even required, the temporary name is just as unique as the final image name eventually is
					// so clashing would occur in both situations
					imageBuild.TemporaryImage = fmt.Sprintf("%s-%s", buildValues.Namespace, composeService) //@TODO maybe get rid of this
					if buildValues.LagoonYAML.Environments[buildValues.Environment].Overrides[composeService].Build.Context != "" {
						imageBuild.Context = buildValues.LagoonYAML.Environments[buildValues.Environment].Overrides[composeService].Build.Context
					}
					// check the dockerfile exists
					if _, err := os.Stat(fmt.Sprintf("%s/%s", imageBuild.Context, imageBuild.DockerFile)); errors.Is(err, os.ErrNotExist) {
						return ServiceValues{}, fmt.Errorf(
							"defined Dockerfile %s for service %s not found",
							fmt.Sprintf("%s/%s", imageBuild.Context, imageBuild.DockerFile), composeService,
						)
					}
				}
			}
			// since we know what the final build image will be, we can set it here, this is what all images will be built as during the build
			// for `pullimages` they will get retagged as this imagename and pushed to the registry
			imageBuild.BuildImage = fmt.Sprintf("%s/%s/%s/%s:%s", buildValues.ImageRegistry, buildValues.Project, buildValues.Environment, composeService, "latest")
			if buildValues.BuildType == "promote" {
				imageBuild.PromoteImage = fmt.Sprintf("%s/%s/%s/%s:%s", buildValues.ImageRegistry, buildValues.Project, buildValues.PromotionSourceEnvironment, composeService, "latest")
			}
			// populate the docker derived information here, this information will be used by the build and pushing scripts
			cService.ImageBuild = &imageBuild

			// // // cService.ImageName = buildValues.ImageReferences[composeService]
			// unfortunately, this uses a specific hash which is computed "AFTER" the image builds take place, so this
			// `ImageName` is an unreliable field in respect to consuming data from generator during phases of a build
			// it would be great if there was a way to precalculate this, but there are other issues that could pop up
			// using the buildname as the tag could be one way, but this could result in container restarts even if the image hash does not change :(
			// for now `ImageName` is disabled, and ImageReferences must be provided whenever templating occurs that needs an image reference
			// luckily the templating engine will reproduce identical data when it is run as to when the image build data is populated as above
			// so when the templating is done in a later step, at least it can be informed of the resulting image references by way of the
			// images flag that is passed to it
			/*
				// example in code
				ImageReferences: map[string]string{
					"myservice": "harbor.example.com/example-project/environment-name/myservice@sha256:abcdefg",
				},

				// in bash, the images are provided from yaml as base64 encoded data to retain formatting
				// the command `lagoon-services` decodes and unmarshals it
				build-deploy-tool template lagoon-services \
					--saved-templates-path ${LAGOON_SERVICES_YAML_FOLDER} \
					--images $(yq3 r -j /kubectl-build-deploy/images.yaml | jq -M -c | base64 -w0)
			*/
		}

		// check if the service has a service port override (this only applies to basic(-persistent))
		servicePortOverride := lagoon.CheckServiceLagoonLabel(composeServiceValues.Labels, "lagoon.service.port")
		if servicePortOverride != "" {
			sPort, err := strconv.Atoi(servicePortOverride)
			if err != nil {
				return ServiceValues{}, fmt.Errorf(
					"The provided service port %s for service %s is not a valid integer: %v",
					servicePortOverride, composeService, err,
				)
			}
			cService.ServicePort = int32(sPort)
		}
		useComposeServices := lagoon.CheckServiceLagoonLabel(composeServiceValues.Labels, "lagoon.service.usecomposeports")
		if useComposeServices == "true" {
			for _, compPort := range composeServiceValues.Ports {
				newService := AdditionalServicePort{
					ServicePort: compPort,
					ServiceName: fmt.Sprintf("%s-%d", composeService, compPort.Target),
				}
				cService.AdditionalServicePorts = append(cService.AdditionalServicePorts, newService)
			}
		}
		return cService, nil
	}
}

// getDBaasEnvironment will check the dbaas provider to see if an environment exists or not
func getDBaasEnvironment(
	buildValues *BuildValues,
	dbaasEnvironment *string,
	lagoonOverrideName,
	lagoonType string,
	debug bool,
) (bool, error) {
	if buildValues.DBaaSEnvironmentTypeOverrides != nil {
		dbaasEnvironmentTypeSplit := strings.Split(buildValues.DBaaSEnvironmentTypeOverrides.Value, ",")
		for _, sType := range dbaasEnvironmentTypeSplit {
			sTypeSplit := strings.Split(sType, ":")
			if sTypeSplit[0] == lagoonOverrideName {
				*dbaasEnvironment = sTypeSplit[1]
			}
		}
	}
	exists, err := buildValues.DBaaSClient.CheckProvider(buildValues.DBaaSOperatorEndpoint, lagoonType, *dbaasEnvironment)
	if err != nil {
		return exists, fmt.Errorf("There was an error checking DBaaS endpoint %s: %v", buildValues.DBaaSOperatorEndpoint, err)
	}
	return exists, nil
}

func checkDuplicateCronjobs(cronjobs []lagoon.Cronjob) error {
	var unique []lagoon.Cronjob
	var duplicates []lagoon.Cronjob
	for _, v := range cronjobs {
		skip := false
		for _, u := range unique {
			if v.Name == u.Name {
				skip = true
				duplicates = append(duplicates, v)
				break
			}
		}
		if !skip {
			unique = append(unique, v)
		}
	}
	var uniqueDuplicates []lagoon.Cronjob
	for _, d := range duplicates {
		for _, u := range unique {
			if d.Name == u.Name {
				uniqueDuplicates = append(uniqueDuplicates, u)
			}
		}
	}
	// join the two together
	result := append(duplicates, uniqueDuplicates...)
	if result != nil {
		b, _ := json.Marshal(result)
		return fmt.Errorf("duplicate named cronjobs detected: %v", string(b))
	}
	return nil
}

// ContainsR checks if a string slice contains a specific string regex match.
func ContainsRegistry(regex []ContainerRegistry, match string) bool {
	for _, v := range regex {
		m, _ := regexp.MatchString(v.URL, match)
		if m {
			return true
		}
	}
	return false
}
