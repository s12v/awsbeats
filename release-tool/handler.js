const octokit = require('@octokit/rest')();
const semver = require('semver');

if (!process.env.GITHUB_TOKEN) {
    throw new Error("GITHUB_TOKEN is required");
}

function run(event, context) {
    Promise.all([
        getLatestRelease(),
        getLatestBeatsRelease(),
    ]).then(results => {
        const [[ourVersion, ourBeatsVersion], beatsVersion] = results;
        if (!semver.valid(ourVersion)) {
            throw new Error(`Invalid version ${ourVersion}`);
        }

        if (!semver.valid(beatsVersion)) {
            throw new Error(`Invalid version ${beatsVersion}`);
        }

        console.log(`Awsbeats: ${ourVersion}-v${ourBeatsVersion}, beats: ${beatsVersion}`);
        if (!semver.valid(ourBeatsVersion) || semver.gt(beatsVersion, ourBeatsVersion)) {
            return doRelease(
                semver.inc(ourVersion, 'patch'),
                semver.clean(beatsVersion)
            ).catch(e => {
                console.error(`Error while creating release: `, e);
                throw e;
            });
        }
    });
}

function getLatestRelease() {
    return octokit.repos.getLatestRelease({owner: 's12v', repo: 'awsbeats'})
        .then(result => result.data.tag_name.split('-v', 2));
}

function getLatestBeatsRelease() {
    return octokit.repos.getLatestRelease({owner: 'elastic', repo: 'beats'})
        .then(result => result.data.tag_name);
}

function doRelease(version, beatsVersion) {
    const tag = `${version}-v${beatsVersion}`;
    console.log(`Releasing "${tag}"`);

    const ok = require('@octokit/rest')();
    ok.authenticate({
        type: 'token',
        token: process.env.GITHUB_TOKEN
    });
    return ok.repos.createRelease({
        owner: 's12v',
        repo: 'awsbeats',
        tag_name: tag,
        name: tag,
        body: `Beats version: v${beatsVersion}`,
        prerelease: true
    })
}

module.exports.run = run;
