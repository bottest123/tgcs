package config

var (
	PathsToJsonFiles = Paths{}
)

type Paths struct {
	PathToTrackedSignals         string
	PathToTelegramConfig         string
	PathToUserConfig             string
	PathToApprovedSignalChannels string
}

func Init(PathsToJsonFilesGlobal string) {
	PathsToJsonFiles.PathToApprovedSignalChannels = PathsToJsonFilesGlobal + "approved_signal_channels.json"
	PathsToJsonFiles.PathToTelegramConfig = PathsToJsonFilesGlobal + "config.json"
	PathsToJsonFiles.PathToUserConfig = PathsToJsonFilesGlobal + "config_.json"
	PathsToJsonFiles.PathToTrackedSignals = PathsToJsonFilesGlobal + "tracked_signals.json"
}
