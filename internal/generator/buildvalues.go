package generator

import "github.com/uselagoon/build-deploy-tool/internal/lagoon"

// BuildValues is the values file data generated by the lagoon build
type BuildValues struct {
	Project              string `json:"project"`
	Environment          string `json:"environment"`
	EnvironmentType      string `json:"environmentType"`
	Namespace            string `json:"namespace"`
	GitSha               string `json:"gitSha"`
	BuildType            string `json:"buildType"`
	Kubernetes           string `json:"kubernetes"`
	LagoonVersion        string `json:"lagoonVersion"`     // this is the version that is bundled in images, probably stop using this?
	LagoonCoreVersion    string `json:"lagoonCoreVersion"` // this is the version that will come from lagoon-core
	ActiveEnvironment    string `json:"activeEnvironment"`
	StandbyEnvironment   string `json:"standbyEnvironment"`
	IsActiveEnvironment  bool   `json:"isActiveEnvironment"`
	IsStandbyEnvironment bool   `json:"isStandbyEnvironment"`
	PodSecurityContext   struct {
		FsGroup    int `json:"fsGroup"`
		RunAsGroup int `json:"runAsGroup"`
		RunAsUser  int `json:"runAsUser"`
	} `json:"podSecurityContext"`
	ImagePullSecrets []struct {
		Name string `json:"name"`
	} `json:"imagePullSecrets"`
	Branch       string `json:"branch"`
	PRNumber     string `json:"prNumber"`
	PRTitle      string `json:"prTitle"`
	PRHeadBranch string `json:"prHeadBranch"`
	PRBaseBranch string `json:"prBaseBranch"`
	Fastly       struct {
		ServiceID     string `json:"serviceId"`
		APISecretName string `json:"apiSecretName"`
		Watch         bool   `json:"watch"`
	} `json:"fastly"`
	FastlyCacheNoCache            string                      `json:"fastlyCacheNoCahce"`
	FastlyAPISecretPrefix         string                      `json:"fastlyAPISecretPrefix"`
	ConfigMapSha                  string                      `json:"configMapSha"`
	Route                         string                      `json:"route"`
	Routes                        []string                    `json:"routes"`
	AutogeneratedRoutes           []string                    `json:"autogeneratedRoutes"`
	RoutesAutogeneratePrefixes    []string                    `json:"routesAutogeneratePrefixes"`
	AutogeneratedRoutesFastly     bool                        `json:"autogeneratedRoutesFastly"`
	Services                      []ServiceValues             `json:"services"`
	Backup                        BackupConfiguration         `json:"backup"`
	Monitoring                    MonitoringConfig            `json:"monitoring"`
	DBaaSOperatorEndpoint         string                      `json:"dbaasOperatorEndpoint"`
	ServiceTypeOverrides          *lagoon.EnvironmentVariable `json:"serviceTypeOverrides"`
	DBaaSEnvironmentTypeOverrides *lagoon.EnvironmentVariable `json:"dbaasEnvironmentTypeOverrides"`
	DBaaSFallbackSingle           bool                        `json:"dbaasFallbackSingle"`
	IngressClass                  string                      `json:"ingressClass"`
	TaskScaleMaxIterations        int                         `json:"taskScaleMaxIterations"`
	TaskScaleWaitTime             int                         `json:"taskScaleWaitTime"`
}

type MonitoringConfig struct {
	Enabled      bool   `json:"enabled"`
	AlertContact string `json:"alertContact"`
	StatusPageID string `json:"statusPageID"`
}

// ServiceValues is the values for a specific service used by a lagoon build
type ServiceValues struct {
	Name                          string                   `json:"name"`         // this is the actual compose service name
	OverrideName                  string                   `json:"overrideName"` // if an override name is provided, use it
	Type                          string                   `json:"type"`
	AutogeneratedRoutesEnabled    bool                     `json:"autogeneratedRoutesEnabled"`
	AutogeneratedRoutesTLSAcme    bool                     `json:"autogeneratedRoutesTLSAcme"`
	AutogeneratedRouteDomain      string                   `json:"autogeneratedRouteDomain"`
	ShortAutogeneratedRouteDomain string                   `json:"shortAutogeneratedRouteDomain"`
	DBaaSEnvironment              string                   `json:"dbaasEnvironment"`
	NativeCronjobs                map[string]CronjobValues `json:"nativeCronjobs"`
	InPodCronjobs                 string                   `json:"inPodCronjobs"`
	ImageName                     string                   `json:"imageName"`
	DeploymentServiceType         string                   `json:"deploymentServiecType"`
}

// CronjobValues is the values for cronjobs
type CronjobValues struct {
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
}

type BackupConfiguration struct {
	PruneRetention PruneRetention              `json:"pruneRetention"`
	PruneSchedule  string                      `json:"pruneSchedule"`
	CheckSchedule  string                      `json:"checkSchedule"`
	BackupSchedule string                      `json:"backupSchedule"`
	S3Endpoint     string                      `json:"s3Endpoint"`
	S3BucketName   string                      `json:"s3BucketName"`
	S3SecretName   string                      `json:"s3SecretName"`
	CustomLocation CustomBackupRestoreLocation `json:"customLocation"`
}

type CustomBackupRestoreLocation struct {
	BackupLocationAccessKey  string `json:"backupLocationAccessKey"`
	BackupLocationSecretKey  string `json:"backupLocationSecretKey"`
	RestoreLocationAccessKey string `json:"restoreLocationAccessKey"`
	RestoreLocationSecretKey string `json:"restoreLocationSecretKey"`
}

type PruneRetention struct {
	Hourly  int `json:"hourly"`
	Daily   int `json:"daily"`
	Weekly  int `json:"weekly"`
	Monthly int `json:"monthly"`
}
