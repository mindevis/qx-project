package mcmanifest

// Velocity JVM flags from PaperMC tuning: https://docs.papermc.io/velocity/tuning/
// Memory (-Xms/-Xmx) is applied separately from the configured RAM.
var velocityJVMFlags = []string{
	"-XX:+AlwaysPreTouch",
	"-XX:+ParallelRefProcEnabled",
	"-XX:+UnlockExperimentalVMOptions",
	"-XX:+UseG1GC",
	"-XX:G1HeapRegionSize=4M",
	"-XX:MaxInlineLevel=15",
}

func VelocityJVMFlags() []string {
	out := make([]string, len(velocityJVMFlags))
	copy(out, velocityJVMFlags)
	return out
}
