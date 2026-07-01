package minecraft

import "path/filepath"

// Launcher data layout:
//
//	{dataDir}/
//	  java/                    shared JRE (only shared game runtime at root)
//	  instances/
//	    {instanceId}/
//	      assets/
//	      libraries/
//	      versions/
//	      cache/               forge installer jars, natives download cache
//	      natives/             extracted native binaries
//	      launcher_profiles.json
//	      saves/, config/, mods/, …  (gameDir — vanilla .minecraft layout)
//	  instances.json           launcher metadata
//	  device_id, device_token, user_auth.json
//
// Existing installs may still have assets/libraries/versions at dataDir root;
// those paths are no longer written — re-launch to populate instance folders.

// InstanceRoot is the base directory for all per-instance game data.
func (d *Downloader) InstanceRoot(instanceID string) string {
	return filepath.Join(d.RootDir, "instances", instanceID)
}

// InstanceGameDir is the Minecraft --gameDir (saves, options, mods, etc.).
func (d *Downloader) InstanceGameDir(instanceID string) string {
	return d.InstanceRoot(instanceID)
}

func (d *Downloader) InstanceAssetsDir(instanceID string) string {
	return filepath.Join(d.InstanceRoot(instanceID), "assets")
}

func (d *Downloader) InstanceLibrariesDir(instanceID string) string {
	return filepath.Join(d.InstanceRoot(instanceID), "libraries")
}

func (d *Downloader) InstanceCacheDir(instanceID string) string {
	return filepath.Join(d.InstanceRoot(instanceID), "cache")
}

func (d *Downloader) InstanceVersionsDir(instanceID, versionKey string) string {
	return filepath.Join(d.InstanceRoot(instanceID), "versions", versionKey)
}

func (d *Downloader) loaderClientJarPath(instanceID, relativePath string) string {
	return filepath.Join(d.InstanceRoot(instanceID), filepath.FromSlash(relativePath))
}
