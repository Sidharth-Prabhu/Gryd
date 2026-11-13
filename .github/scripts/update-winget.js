module.exports = async ({ github, context, core }) => {
  const version = process.env.GRYD_VERSION;
  const tag = process.env.GRYD_TAG;
  const winAmd64Sha = process.env.WIN_AMD64_SHA;
  const winArm64Sha = process.env.WIN_ARM64_SHA;
  const today = new Date().toISOString().split('T')[0];

  const versionYaml = `# yaml-language-server: $schema=https://aka.ms/winget-manifest.version.1.6.0.schema.json
PackageIdentifier: SidharthPrabhu.Gryd
PackageVersion: ${version}
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.6.0
`;

  const localeYaml = `# yaml-language-server: $schema=https://aka.ms/winget-manifest.defaultLocale.1.6.0.schema.json
PackageIdentifier: SidharthPrabhu.Gryd
PackageVersion: ${version}
PackageLocale: en-US
Publisher: Sidharth Prabhu
PublisherUrl: https://github.com/Sidharth-Prabhu
PublisherSupportUrl: https://github.com/Sidharth-Prabhu/gryd/issues
Author: Sidharth Prabhu
PackageName: Gryd
PackageUrl: https://github.com/Sidharth-Prabhu/gryd
License: MIT
LicenseUrl: https://github.com/Sidharth-Prabhu/gryd/blob/main/LICENSE
ShortDescription: Terminal UI and development orchestrator for managing multi-service fullstack applications.
Description: |-
  Gryd is a cross-platform multi-service development orchestrator and interactive TUI dashboard.
  It provides real-time logs, process control, health monitoring, resource usage metrics, and Cloudflare tunnel sharing.
Moniker: gryd
Tags:
  - cli
  - developer-tools
  - microservices
  - orchestration
  - terminal
  - tui
ReleaseNotesUrl: https://github.com/Sidharth-Prabhu/gryd/releases/tag/${tag}
ManifestType: defaultLocale
ManifestVersion: 1.6.0
`;

  const installerYaml = `# yaml-language-server: $schema=https://aka.ms/winget-manifest.installer.1.6.0.schema.json
PackageIdentifier: SidharthPrabhu.Gryd
PackageVersion: ${version}
InstallerLocale: en-US
InstallerType: zip
ReleaseDate: "${today}"
Installers:
  - Architecture: x64
    NestedInstallerType: portable
    NestedInstallerFiles:
      - RelativeFilePath: gryd.exe
        PortableCommandAlias: gryd
    InstallerUrl: https://github.com/Sidharth-Prabhu/gryd/releases/download/${tag}/gryd_windows_amd64.zip
    InstallerSha256: ${winAmd64Sha}
    UpgradeBehavior: uninstallPrevious
  - Architecture: arm64
    NestedInstallerType: portable
    NestedInstallerFiles:
      - RelativeFilePath: gryd.exe
        PortableCommandAlias: gryd
    InstallerUrl: https://github.com/Sidharth-Prabhu/gryd/releases/download/${tag}/gryd_windows_arm64.zip
    InstallerSha256: ${winArm64Sha}
    UpgradeBehavior: uninstallPrevious
ManifestType: installer
ManifestVersion: 1.6.0
`;

  const forkOwner = 'Sidharth-Prabhu';
  const forkRepo = 'winget-pkgs';
  const upstreamOwner = 'microsoft';
  const upstreamRepo = 'winget-pkgs';
  const branchName = `sidharthprabhu-gryd-${version}`;
  const basePath = `manifests/s/SidharthPrabhu/Gryd/${version}`;

  // 1. Get default branch SHA of fork
  const { data: refData } = await github.rest.git.getRef({
    owner: forkOwner,
    repo: forkRepo,
    ref: 'heads/master',
  });
  const masterSha = refData.object.sha;

  // 2. Create branch on fork
  try {
    await github.rest.git.createRef({
      owner: forkOwner,
      repo: forkRepo,
      ref: `refs/heads/${branchName}`,
      sha: masterSha,
    });
    console.log(`Created branch ${branchName}`);
  } catch (e) {
    console.log(`Branch ${branchName} already exists or error:`, e.message);
  }

  // 3. Commit the 3 manifest files
  const files = [
    { path: `${basePath}/SidharthPrabhu.Gryd.yaml`, content: versionYaml },
    { path: `${basePath}/SidharthPrabhu.Gryd.locale.en-US.yaml`, content: localeYaml },
    { path: `${basePath}/SidharthPrabhu.Gryd.installer.yaml`, content: installerYaml }
  ];

  for (const f of files) {
    let sha = undefined;
    try {
      const { data } = await github.rest.repos.getContent({
        owner: forkOwner,
        repo: forkRepo,
        path: f.path,
        ref: branchName,
      });
      sha = data.sha;
    } catch (e) {
      // File does not exist yet
    }

    await github.rest.repos.createOrUpdateFileContents({
      owner: forkOwner,
      repo: forkRepo,
      path: f.path,
      message: `New package: SidharthPrabhu.Gryd version ${version}`,
      content: Buffer.from(f.content).toString('base64'),
      branch: branchName,
      sha: sha,
    });
    console.log(`Uploaded ${f.path}`);
  }

  // 4. Open Pull Request on microsoft/winget-pkgs
  try {
    const { data: pr } = await github.rest.pulls.create({
      owner: upstreamOwner,
      repo: upstreamRepo,
      title: `New package: SidharthPrabhu.Gryd version ${version}`,
      head: `${forkOwner}:${branchName}`,
      base: 'master',
      body: `### Package Identifier: SidharthPrabhu.Gryd\n### Package Version: ${version}\n\nAutomated package release submission for Gryd ${version}.`,
    });
    console.log(`PR created: ${pr.html_url}`);
  } catch (e) {
    console.log('PR creation response / note:', e.message);
  }
};
