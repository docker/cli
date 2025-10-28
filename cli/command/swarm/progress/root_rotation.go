package progress

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/moby/moby/client/pkg/progress"
	"github.com/moby/moby/client/pkg/streamformatter"
	"github.com/opencontainers/go-digest"
)

const (
	certsRotatedStr = "  rotated TLS certificates"
	rootsRotatedStr = "  rotated CA certificates"
	// rootsAction has a single space because rootsRotatedStr is one character shorter than certsRotatedStr.
	// This makes sure the progress bar are aligned.
	certsAction = ""
	rootsAction = " "
)

// RootRotationProgress outputs progress information for convergence of a root rotation.
func RootRotationProgress(ctx context.Context, apiClient client.APIClient, progressWriter io.WriteCloser) error {
	defer func() {
		_ = progressWriter.Close()
	}()

	progressOut := streamformatter.NewJSONProgressOutput(progressWriter, false)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	defer signal.Stop(sigint)

	// draw 2 progress bars, 1 for nodes with the correct cert, 1 for nodes with the correct trust root
	progress.Update(progressOut, "desired root digest", "")
	progress.Update(progressOut, certsRotatedStr, certsAction)
	progress.Update(progressOut, rootsRotatedStr, rootsAction)

	var done bool

	for {
		info, err := apiClient.SwarmInspect(ctx, client.SwarmInspectOptions{})
		if err != nil {
			return err
		}

		if done {
			return nil
		}

		res, err := apiClient.NodeList(ctx, client.NodeListOptions{})
		if err != nil {
			return err
		}

		done = updateProgress(progressOut, info.Swarm.ClusterInfo.TLSInfo, res.Items, info.Swarm.ClusterInfo.RootRotationInProgress)

		select {
		case <-time.After(200 * time.Millisecond):
		case <-sigint:
			if !done {
				progress.Message(progressOut, "", "Operation continuing in background.")
				progress.Message(progressOut, "", "Use `swarmctl cluster inspect default` to check progress.")
			}
			return nil
		}
	}
}

func updateProgress(progressOut progress.Output, desiredTLSInfo swarm.TLSInfo, nodes []swarm.Node, rootRotationInProgress bool) bool {
	// write the current desired root cert's digest, because the desired root certs might be too long
	_ = progressOut.WriteProgress(progress.Progress{
		ID:     "desired root digest",
		Action: digest.FromBytes([]byte(desiredTLSInfo.TrustRoot)).String(),
	})

	// If we had reached a converged state, check if we are still converged.
	var certsRight, trustRootsRight int64
	for _, n := range nodes {
		if bytes.Equal(n.Description.TLSInfo.CertIssuerPublicKey, desiredTLSInfo.CertIssuerPublicKey) &&
			bytes.Equal(n.Description.TLSInfo.CertIssuerSubject, desiredTLSInfo.CertIssuerSubject) {
			certsRight++
		}

		if n.Description.TLSInfo.TrustRoot == desiredTLSInfo.TrustRoot {
			trustRootsRight++
		}
	}

	total := int64(len(nodes))
	_ = progressOut.WriteProgress(progress.Progress{
		ID:      certsRotatedStr,
		Action:  certsAction,
		Current: certsRight,
		Total:   total,
		Units:   "nodes",
	})

	rootsProgress := progress.Progress{
		ID:      rootsRotatedStr,
		Action:  rootsAction,
		Current: trustRootsRight,
		Total:   total,
		Units:   "nodes",
	}

	if certsRight == total && !rootRotationInProgress {
		_ = progressOut.WriteProgress(rootsProgress)
		return certsRight == total && trustRootsRight == total
	}

	// we still have certs that need renewing, so display that there are zero roots rotated yet
	rootsProgress.Current = 0
	_ = progressOut.WriteProgress(rootsProgress)
	return false
}
