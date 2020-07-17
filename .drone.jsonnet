local tags(spin) = {
    name: 'tags-%(spin)s' % { spin: spin },
    image: 'bitnami/git',
    commands: [
      "git fetch --tags",
      'echo "${DRONE_BRANCH/\//-}-$(git describe --tags --always)-%(spin)s" > .tags' % { spin: spin },
    ],
};



local docker(spin)= {
    name: 'publish-%(spin)s' % { spin: spin },
    image: 'plugins/docker',
    settings: {
      dockerfile: "Dockerfile",
      repo: "${CI_REPO}",
      username: "digtux",
      password: { from_secret: "dockerhub-pass"}
      },
  };


{
  kind: "pipeline",
  name: "build",

  steps: [

    tags('alpine'),
    docker('alpine'),

    tags('kapitan'),
    docker('kapitan'),

  ],
  trigger:{
    branch: [
      "master",
      "feature/*",
    ],
    event: [
      "push",
    ],
  },

}
