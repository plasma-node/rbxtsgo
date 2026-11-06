import Log, { Logger, LogLevel } from "@rbxts/log";

Log.SetLogger(
	Logger.configure()
		.SetMinLogLevel(LogLevel.Debugging)
		.WriteTo(Log.RobloxOutput())
		.Create(),
);

class Service {
	private readonly logger = Log.ForContext(Service);
	run(value: number) {
		this.logger.Info("Running {Value}", value);
		this.logger.Warn("Careful");
	}
}

export function main() {
	Log.Info("Starting {Name}", "demo");
	new Service().run(1);
}
