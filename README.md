# Laminar

## Don't let your deployments be turbulent!


## What is this?

Laminar is a reasonably simple `GitOps` tool, inspired by things like [flux](https://github.com/fluxcd/flux) but with a very different architecture.

The TL;DR of what it does is this:

Step 1, you tell it about your docker registry(s):
```yaml
dockerRegistries:
- reg: 1122334455.dkr.ecr.eu-west-2.amazonaws.com/acmecorp
  name: acmecorp-aws
```

Step 2, you give it (git+ssh) access to your git repo:
```yaml
git:
- name: manifests
  url: gitoperations@github.com:acmecorp/k8s-manifests.gitoperations
  branch: master
  key: ~/example_ssh_id_rsa  # path to the SSH key (needed for gitoperations)
  pollFreq: 120              # How often to sync..
  updates: []                # list of updates (see next step)
```

Step 3, you tell laminar which files you want it to operate on (in the same git repo)
```yaml
git:
- name: manifests
  updates:

  - pattern: "glob:develop-*" # will match docker tags such as "develop-1.2" or "develop-short_sha"
    # where to look in your gitoperations repo
    files:
    - dev/  # laminar will search in directory "dev"
```

Step 4, run laminar
```shell
AWS_PROFILE=myprofile ./laminar
```

Laminar will then do the following:
- clone the git repo `k8s-manifests` and checkout branch `master`
- search for files under `dev/` inside your git repo
- inside the files search for strings such as: `1122334455.dkr.ecr.eu-west-2.amazonaws.com/acmecorp/app-name:develop-52af76b8` (any tag that matches `develop-*`)
- index the tags available from `1122334455.dkr.ecr.eu-west-2.amazonaws.com/acmecorp/app-name`
- check if there are any tags (matching `develop-*`) which are more recent than `develop-52af76b8`
- a. if there is a more recent tag in ECR (eg: `develop-b2ee56fd`) update the file from  `develop-52af76b8` -> `develop-b2ee56fd`
- b: git commit the updated file(s), then git push
- sleep until the end of `pollFreq`
- repeat


# Reasoning
We love weave flux.. but it makes working with templated manifests challenging. If you're running 10x kubernetes clusters it also makes very little sense to have each one polling your docker registries.

- With this pattern you only need one tool to automate your GitOps.
- This is also compatible with docker-compose or practically any text file really.
- easier to co-ordinate many changes at the same time.
- Lets assume my manifests are templated in my git repo.. flux would want to patch the file that is the output of my templating. With laminar you can update both, or just one. This opens the door to using `kapitan`, `tanka`, `cuelang` or any templating really.

# Features

- [x] checkout your git repo(s)
- [x] search yaml files looking for obvious docker images
- [x] check if there are more recent docker tags are in your registry and ready to be deployed
- [x] update your git repo with the more recent tags
- [x] dynamically load a list of files and image:tag patterns from the remote git repos (`.laminar.yaml`)
- [ ] add `exec` action so commands can be run after modifying git (and before the `git commit`)
- [ ] exclude list (users can blacklist promoting specific image+tag patterns)
- [ ] docker image + deployment manifest
- [ ] prometheus metrics


# issues/todo
- [ ] after initialCheckout(), if the (remote) git repo is reverted with a `--force` push we should handle that and re-clone
- [ ] more tests, do this when refactoring the logic
- [ ] the main loop is currently (MVP) and simply just a `time.Sleep()`. There is no concurrnecy/`time.Tick()` yet.
- [ ] occasional errors from registry polling not surfacing correctly. probably needs some attention.
- [ ] tidy up (specifically the business logic around change requests and add maybe add some concurrency)
- [ ] quick start/tutorial/example docs!
- [ ] example: PrometheusAlerts
- [ ] example: grafana dashboard


## optional behaviours
- [x] glob filters on tags (eg `master-*` )
- [x] only operate on specific files or directories in your git repo
- [x] multiple git repos (not tested well)
- [ ] built in "post sync" `actions` such as: "slack alert", "github PR"
- [ ] user configurable `actions` such as running a shell script to re-render charts
- [ ] user adjustable file exclude list
- [ ] an api endpoint that can trigger a sync (so your CI can hit it after pushing a new image)
- [ ] a simple gui with some info about tags and images
- [ ] individual auth configuration available for registries (allowing support for multiple GCR and ECR)
- [ ] other tag matching patterns, specifically: `semver` and `regex`
