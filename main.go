package main

import (
	"iwals/internal/agent"
	"iwals/internal/config"
	"log"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	cli := &cli{}

	cmd := &cobra.Command{
		Use:     "iwals",
		PreRunE: cli.setupConfig,
		RunE:    cli.run,
	}

	if err := setupFlags(cmd); err != nil {
		log.Fatal(err)
	}
}

type cli struct {
	cfg cfg
}

type cfg struct {
	agent.Config
	ServerTLSConfig config.TLSConfig
	PeerTLSConfig   config.TLSConfig
}

func setupFlags(cmd *cobra.Command) error {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Flags().String("config-file", "", "Path to config file.")

	dataDir := path.Join(os.TempDir(), "iwals")
	cmd.Flags().String("data-dir", dataDir, "Directory to store log and Raft data.")
	cmd.Flags().String("node-name", hostname, "Unique server ID.")
	cmd.Flags().String("bind-addr", "127.0.0.1:8401", "Address to bind Serf")
	cmd.Flags().Int("rpc-port", 8400, "Port for RPC client (and Raft) connections")
	cmd.Flags().StringSlice("start-join-addrs", nil, "Serf addresses to join.")
	cmd.Flags().Bool("bootstrap", false, "Bootstrap the cluster.")
	cmd.Flags().String("acl-model-file", "", "Path to ACL model.")
	cmd.Flags().String("acl-policy-file", "", "Path to ACL policy file.")

	cmd.Flags().String("server-tls-cert-file", "", "Path to server tls certificate file.")
	cmd.Flags().String("server-tls-key-file", "", "Path to server tls key file.")
	cmd.Flags().String("server-tls-ca-file", "", "Path to server certificate authority.")
	cmd.Flags().String("peer-tls-cert-file", "", "Path to peer tls certificate file.")
	cmd.Flags().String("peer-tls-key-file", "", "Path to peer tls key file.")
	cmd.Flags().String("peer-tls-ca-file", "", "Path to peer tls certificate authority file.")

	return viper.BindPFlags(cmd.Flags())
}

func (c *cli) setupConfig(cmd *cobra.Command, agrs []string) error {
	var err error
	configFile, err := cmd.Flags().GetString("config-file")
	if err != nil {
		return err
	}
	viper.SetConfigFile(configFile)

	if err = viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return err
		}
	}

	c.cfg.DataDir, err = cmd.Flags().GetString("data-dir")
	if err != nil {
		return err
	}
	c.cfg.NodeName, err = cmd.Flags().GetString("node-name")
	if err != nil {
		return err
	}
	c.cfg.BindAddr, err = cmd.Flags().GetString("bind-addr")
	if err != nil {
		return err
	}

	c.cfg.RPCPort, err = cmd.Flags().GetInt("rpc-port")
	if err != nil {
		return err
	}
	c.cfg.StartJoinAddrs, err = cmd.Flags().GetStringSlice("start-join-addrs")
	if err != nil {
		return err
	}
	c.cfg.Bootstrap, err = cmd.Flags().GetBool("bootstrap")
	if err != nil {
		return err
	}

	c.cfg.ACLModelFile, err = cmd.Flags().GetString("acl-model-file")
	if err != nil {
		return err
	}
	c.cfg.ACLPolicyFile, err = cmd.Flags().GetString("acl-policy-file")
	if err != nil {
		return err
	}
	c.cfg.ServerTLSConfig.CAFile, err = cmd.Flags().GetString("server-tls-ca-file")
	if err != nil {
		return err
	}
	c.cfg.ServerTLSConfig.CertFile, err = cmd.Flags().GetString("server-tls-cert-file")
	if err != nil {
		return err
	}
	c.cfg.ServerTLSConfig.KeyFile, err = cmd.Flags().GetString("server-tls-key-file")
	if err != nil {
		return err
	}

	c.cfg.PeerTLSConfig.CAFile, err = cmd.Flags().GetString("peer-tls-ca-file")
	if err != nil {
		return err
	}
	c.cfg.PeerTLSConfig.CertFile, err = cmd.Flags().GetString("peer-tls-cert-file")
	if err != nil {
		return err
	}
	c.cfg.PeerTLSConfig.KeyFile, err = cmd.Flags().GetString("peer-tls-key-file")
	if err != nil {
		return err
	}

	if c.cfg.ServerTLSConfig.CertFile != "" &&
		c.cfg.ServerTLSConfig.KeyFile != "" {
		c.cfg.ServerTLSConfig.Server = true
		c.cfg.Config.ServerTLSConfig, err = config.SetupTLSConfig(c.cfg.ServerTLSConfig)
		if err != nil {
			return err
		}
	}

	if c.cfg.PeerTLSConfig.CertFile != "" &&
		c.cfg.PeerTLSConfig.KeyFile != "" {
		c.cfg.Config.PeerTLSConfig, err = config.SetupTLSConfig(c.cfg.PeerTLSConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *cli) run(cmd *cobra.Command, args []string) error {
	var err error
	agent, err := agent.New(c.cfg.Config)
	if err != nil {
		return err
	}
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	<-sigc
	return agent.Shutdown()
}
