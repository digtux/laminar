package cfg

// Global settings such as git commit user/email
type Global struct {
	GitUser    string       `yaml:"gitUser"`
	GitEmail   string       `yaml:"gitEmail"`
	GitMessage interface{}  `yaml:"gitMessage"`
	GitHubToken string      `yaml:"gitHubToken"`
}


// Config is the top level of config
type Config struct {
	DockerRegistries []DockerRegistry `yaml:"dockerRegistries"`
	GitRepos         []GitRepo        `yaml:"git"`
	Global           Global           `yaml:"global"`
}

// DockerRegistry contains info about the docker registries
type DockerRegistry struct {
	Reg     string `yaml:"reg"`
	Name    string `yaml:"name"`
	TimeOut int    `yaml:"timeOut,omitempty"`
}

type BlackList struct {
	Pattern string `yaml:"pattern"`
}

// GitRepo which laminar operates on
type GitRepo struct {
	URL               string    `yaml:"url"`
	Branch            string    `yaml:"branch"`
	Key               string    `yaml:"key"`
	PollFreq          int       `yaml:"pollFreq"`
	Name              string    `yaml:"name"`
	RemoteConfig      bool      `yaml:"remoteConfig"` // propogate []Updates from remote git ".laminar.yaml" ?
	Updates           []Updates `yaml:"updates,omitempty"`
	PreCommitCommands []string  `yaml:"preCommitCommands,omitempty"`
	//PostChange   []PostChanges `yaml:"postChange"`
}

//// PostChanges to do after updating a gitrepo
//type PostChanges struct {
//	Action string `yaml:"action"`
//	Data   string `yaml:"data"`
//}

// Files to operate upon in a git repo
type Files struct {
	Path string `yaml:"path"`
}

// Update contains instructions about what to do with matching image
type Updates struct {
	PatternString string      `yaml:"pattern"`
	Files         []Files     `yaml:"files"`
	BlackList     []BlackList `yaml:"blacklist"`
}

type RemoteUpdates struct {
	Updates []Updates `yaml:"updates"`
}
