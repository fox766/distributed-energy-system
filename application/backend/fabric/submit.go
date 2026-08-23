package fabric

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/grpc/status"
)

// writeMu serializes ledger writes from this backend.
//
// Fabric validates optimistically: two concurrent transactions that touch the
// same key both endorse successfully and then race to commit — the loser gets
// an MVCC conflict and must re-run. Retrying helps, but under a burst (15
// concurrent GenerateEnergy calls on one producer measured 5 exhausted
// retries), the losers keep colliding. Serializing writes here makes
// single-process conflicts structurally impossible while read queries
// (EvaluateTransaction) stay fully concurrent. The write throughput of this
// system — two schedulers at a few requests per minute plus user actions —
// is far below anything this mutex would bottleneck.
var writeMu sync.Mutex

// submitAttempts is the number of times Submit tries a transaction before
// giving up. Retries only happen for read/write-set conflicts. Eight attempts
// with the backoff below survive a burst of ~10 concurrent writers on the same
// key; four was not enough under that load (measured: 6/10 requests still
// exhausted their retries).
const submitAttempts = 8

// maxRetryBackoff caps the wait between conflict retries. Conflicts clear as
// soon as the competing transaction commits — typically tens of milliseconds —
// but when many transactions queue on one key, the loser needs progressively
// longer sleeps to slot into the serialized commit order.
const maxRetryBackoff = 2 * time.Second

// Submit submits a transaction, retrying when it fails to commit because of a
// read/write-set conflict.
//
// Fabric validates transactions optimistically: a transaction that read a key
// which another transaction wrote in the meantime is rejected at commit time
// with MVCC_READ_CONFLICT (or PHANTOM_READ_CONFLICT for range queries). This is
// expected under concurrency, not a bug — the background generation scheduler
// and a user action routinely touch the same user record. The correct response
// is to re-run the transaction against fresh state, which is what retrying does.
//
// Only conflict errors are retried. Chaincode rejections (insufficient balance,
// bad status, …) are deterministic and returned immediately.
func (g *Gateway) Submit(name string, args ...string) ([]byte, error) {
	writeMu.Lock()
	defer writeMu.Unlock()

	var lastErr error
	for attempt := 1; attempt <= submitAttempts; attempt++ {
		result, err := g.Contract.SubmitTransaction(name, args...)
		if err == nil {
			if attempt > 1 {
				log.Printf("fabric: %s succeeded on attempt %d after write conflict", name, attempt)
			}
			return result, nil
		}
		lastErr = err
		if !isConflict(err) {
			return nil, err
		}
		if attempt < submitAttempts {
			// Exponential backoff with a cap; see maxRetryBackoff. Conflicts now
			// only come from other processes or from peers, so these waits are
			// rarely exercised.
			backoff := time.Duration(300) * time.Millisecond << (attempt - 1)
			if backoff > maxRetryBackoff {
				backoff = maxRetryBackoff
			}
			time.Sleep(backoff)
		}
	}
	return nil, fmt.Errorf("%s failed after %d attempts: %w", name, submitAttempts, lastErr)
}

// isConflict reports whether err is a read/write-set conflict worth retrying.
func isConflict(err error) bool {
	var commitErr *client.CommitError
	if errors.As(err, &commitErr) {
		switch commitErr.Code {
		case peer.TxValidationCode_MVCC_READ_CONFLICT,
			peer.TxValidationCode_PHANTOM_READ_CONFLICT:
			return true
		}
		return false
	}
	// Endorsement-time conflicts surface as plain messages rather than a typed
	// CommitError, so fall back to matching the validation code names.
	msg := err.Error()
	return strings.Contains(msg, "MVCC_READ_CONFLICT") ||
		strings.Contains(msg, "PHANTOM_READ_CONFLICT")
}

// ErrorDetail unwraps a fabric-gateway error down to the message the chaincode
// actually returned.
//
// The gateway reports endorsement failures as a gRPC status whose message is
// the useless "failed to endorse transaction, see attached details for more
// info"; the real reason (e.g. "insufficient balance: have ¥0.00, need ¥2.00")
// travels in the status details as a gateway.ErrorDetail per endorsing peer.
// Without this, every chaincode rejection reaches the client as the same
// opaque sentence.
func ErrorDetail(err error) string {
	if err == nil {
		return ""
	}

	// Endorsement/submit/commit-status failures are gRPC status errors whose
	// details carry the per-peer chaincode message.
	if st, ok := status.FromError(err); ok {
		var msgs []string
		seen := make(map[string]bool)
		for _, d := range st.Details() {
			detail, ok := d.(*gateway.ErrorDetail)
			if !ok || detail.GetMessage() == "" {
				continue
			}
			m := cleanChaincodeMessage(detail.GetMessage())
			if !seen[m] {
				seen[m] = true
				msgs = append(msgs, m)
			}
		}
		if len(msgs) > 0 {
			return strings.Join(msgs, "; ")
		}
	}

	// A transaction that endorsed but failed validation (including conflicts
	// exhausted by Submit) has no details — its own message names the code.
	var commitErr *client.CommitError
	if errors.As(err, &commitErr) {
		return commitErr.Error()
	}

	return err.Error()
}

// cleanChaincodeMessage strips the wrapper the peer puts around a chaincode
// error so the caller sees the message the contract wrote.
func cleanChaincodeMessage(msg string) string {
	// Typical shape:
	//   chaincode response 500, insufficient balance: have ¥0.00, need ¥2.00
	if i := strings.Index(msg, "chaincode response "); i >= 0 {
		if j := strings.Index(msg[i:], ", "); j >= 0 {
			return strings.TrimSpace(msg[i+j+2:])
		}
	}
	// Or a bare gRPC status line.
	if i := strings.Index(msg, "desc = "); i >= 0 {
		return strings.TrimSpace(msg[i+len("desc = "):])
	}
	return strings.TrimSpace(msg)
}
