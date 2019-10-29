package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kubernetes-csi/csi-lib-iscsi/iscsi"
	"github.com/openebs/jiva-csi/pkg/config"
	"github.com/openebs/jiva-csi/pkg/driver"
	"github.com/openebs/jiva-csi/pkg/kubernetes/client"
	"github.com/openebs/jiva-csi/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	k8scfg "sigs.k8s.io/controller-runtime/pkg/client/config"
)

// log2LogrusWriter implement io.Writer interface used to enable
// debug logs for iscsi lib
type log2LogrusWriter struct {
	entry *logrus.Entry
}

// Write redirects the std log to logrus
func (w *log2LogrusWriter) Write(b []byte) (int, error) {
	n := len(b)
	if n > 0 && b[n-1] == '\n' {
		b = b[:n-1]
	}
	w.entry.Debug(string(b))
	return n, nil
}

/*
 * main routine to start the jiva-csi-driver. The same
 * binary is used for controller and agent deployment.
 * they both are differentiated via plugin command line
 * argument. To start the controller, we have to pass
 * --plugin=controller and to start it as agent, we have
 * to pass --plugin=agent.
 */
func main() {
	_ = flag.CommandLine.Parse([]string{})
	var config = config.Default()

	var enableISCSIDebug bool

	cmd := &cobra.Command{
		Use:   "jiva-csi-driver",
		Short: "driver for provisioning jiva volume",
		Long:  `provisions and deprovisions the volume`,
		Run: func(cmd *cobra.Command, args []string) {
			run(config)
		},
	}

	cmd.Flags().AddGoFlagSet(flag.CommandLine)

	cmd.PersistentFlags().StringVar(
		&config.NodeID, "nodeid", "", "NodeID to identify the node running this driver",
	)

	cmd.PersistentFlags().StringVar(
		&config.Version, "version", "", "Displays driver version",
	)

	cmd.PersistentFlags().StringVar(
		&config.Endpoint, "endpoint", "unix://csi/csi.sock", "CSI endpoint",
	)

	cmd.PersistentFlags().StringVar(
		&config.DriverName, "name", "jiva-csi-driver", "Name of this driver",
	)

	cmd.PersistentFlags().StringVar(
		&config.PluginType, "plugin", "jiva-csi-plugin", "Type of this driver i.e. controller or node",
	)

	cmd.Flags().BoolVarP(
		&enableISCSIDebug, "enableISCSIDebug", "d", false, "Enable iscsi debug logs",
	)

	if config.PluginType == "agent" && enableISCSIDebug {
		logrus.SetLevel(logrus.DebugLevel)
		iscsi.EnableDebugLogging(&log2LogrusWriter{
			entry: logrus.StandardLogger().WithField("logger", "iscsi"),
		})
	}

	err := cmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(1)
	}
}

func run(config *config.Config) {
	if config.Version == "" {
		config.Version = version.Version
	}

	logrus.Infof("%s - %s", version.Version, version.Commit)
	logrus.Infof(
		"DriverName: %s Plugin: %s EndPoint: %s NodeID: %s",
		config.DriverName,
		config.PluginType,
		config.Endpoint,
		config.NodeID,
	)

	// get the kube config
	cfg, err := k8scfg.GetConfig()
	if err != nil {
		logrus.Fatalf("error getting config: %v", err)
	}

	// generate a new client object
	cli, err := client.New(cfg)
	if err != nil {
		logrus.Fatalf("error creating client from config: %v", err)
	}

	if err := cli.RegisterAPI(); err != nil {
		logrus.Fatalf("error registering API: %v", err)
	}

	err = driver.New(config, cli).Run()
	if err != nil {
		log.Fatalln(err)
	}
	os.Exit(0)
}
