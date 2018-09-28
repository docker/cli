package commands

import (
	"os"
	"runtime"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/builder"
	"github.com/docker/cli/cli/command/checkpoint"
	"github.com/docker/cli/cli/command/config"
	"github.com/docker/cli/cli/command/container"
	"github.com/docker/cli/cli/command/engine"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/cli/command/manifest"
	"github.com/docker/cli/cli/command/network"
	"github.com/docker/cli/cli/command/node"
	"github.com/docker/cli/cli/command/plugin"
	"github.com/docker/cli/cli/command/registry"
	"github.com/docker/cli/cli/command/secret"
	"github.com/docker/cli/cli/command/service"
	"github.com/docker/cli/cli/command/stack"
	"github.com/docker/cli/cli/command/swarm"
	"github.com/docker/cli/cli/command/system"
	"github.com/docker/cli/cli/command/trust"
	"github.com/docker/cli/cli/command/volume"
	"github.com/spf13/cobra"
)

// AddCommands adds all the commands from cli/command to the root command
// 将 cli/command 包中的所有命令添加到根命令
func AddCommands(cmd *cobra.Command, dockerCli command.Cli) {
	cmd.AddCommand(
		// checkpoint/容器的检查点
		checkpoint.NewCheckpointCommand(dockerCli),

		// config/Docker配置
		config.NewConfigCommand(dockerCli),

		// container/容器
		container.NewContainerCommand(dockerCli),
		container.NewRunCommand(dockerCli),

		// image/镜像
		image.NewImageCommand(dockerCli),
		image.NewBuildCommand(dockerCli),

		// builder/构建
		builder.NewBuilderCommand(dockerCli),

		// manifest/Docker镜像清单和清单列表
		manifest.NewManifestCommand(dockerCli),

		// network/网络
		network.NewNetworkCommand(dockerCli),

		// node/节点
		node.NewNodeCommand(dockerCli),

		// plugin/插件
		plugin.NewPluginCommand(dockerCli),

		// registry/Docker注册中心
		registry.NewLoginCommand(dockerCli),
		registry.NewLogoutCommand(dockerCli),
		registry.NewSearchCommand(dockerCli),

		// secret/Docker秘钥
		secret.NewSecretCommand(dockerCli),

		// service/服务
		service.NewServiceCommand(dockerCli),

		// system/Docker系统
		system.NewSystemCommand(dockerCli),
		system.NewVersionCommand(dockerCli),

		// stack/Docker堆栈
		stack.NewStackCommand(dockerCli),
		stack.NewTopLevelDeployCommand(dockerCli),

		// swarm
		swarm.NewSwarmCommand(dockerCli),

		// trust/Docker镜像上的证书
		trust.NewTrustCommand(dockerCli),

		// volume/磁盘存储
		volume.NewVolumeCommand(dockerCli),

		// legacy commands may be hidden/遗留命令可能被隐藏
		// system/Docker系统
		hide(system.NewEventsCommand(dockerCli)),
		hide(system.NewInfoCommand(dockerCli)),
		hide(system.NewInspectCommand(dockerCli)),
		// container/容器
		hide(container.NewAttachCommand(dockerCli)),
		hide(container.NewCommitCommand(dockerCli)),
		hide(container.NewCopyCommand(dockerCli)),
		hide(container.NewCreateCommand(dockerCli)),
		hide(container.NewDiffCommand(dockerCli)),
		hide(container.NewExecCommand(dockerCli)),
		hide(container.NewExportCommand(dockerCli)),
		hide(container.NewKillCommand(dockerCli)),
		hide(container.NewLogsCommand(dockerCli)),
		hide(container.NewPauseCommand(dockerCli)),
		hide(container.NewPortCommand(dockerCli)),
		hide(container.NewPsCommand(dockerCli)),
		hide(container.NewRenameCommand(dockerCli)),
		hide(container.NewRestartCommand(dockerCli)),
		hide(container.NewRmCommand(dockerCli)),
		hide(container.NewStartCommand(dockerCli)),
		hide(container.NewStatsCommand(dockerCli)),
		hide(container.NewStopCommand(dockerCli)),
		hide(container.NewTopCommand(dockerCli)),
		hide(container.NewUnpauseCommand(dockerCli)),
		hide(container.NewUpdateCommand(dockerCli)),
		hide(container.NewWaitCommand(dockerCli)),
		// image/镜像
		hide(image.NewHistoryCommand(dockerCli)),
		hide(image.NewImagesCommand(dockerCli)),
		hide(image.NewImportCommand(dockerCli)),
		hide(image.NewLoadCommand(dockerCli)),
		hide(image.NewPullCommand(dockerCli)),
		hide(image.NewPushCommand(dockerCli)),
		hide(image.NewRemoveCommand(dockerCli)),
		hide(image.NewSaveCommand(dockerCli)),
		hide(image.NewTagCommand(dockerCli)),
	)
	if runtime.GOOS == "linux" {
		// engine/本地docker引擎
		cmd.AddCommand(engine.NewEngineCommand(dockerCli))
	}
}

func hide(cmd *cobra.Command) *cobra.Command {
	// If the environment variable with name "DOCKER_HIDE_LEGACY_COMMANDS" is not empty,
	// these legacy commands (such as `docker ps`, `docker exec`, etc)
	// will not be shown in output console.
	// 这些遗留命令不会显示在输出控制台中
	if os.Getenv("DOCKER_HIDE_LEGACY_COMMANDS") == "" {
		return cmd
	}
	cmdCopy := *cmd
	cmdCopy.Hidden = true
	cmdCopy.Aliases = []string{}
	return &cmdCopy
}
