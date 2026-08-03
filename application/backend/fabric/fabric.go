package fabric

import (
	"crypto/x509"
	"fmt"
	"os"
	"path"
	"time"

	"backend/config"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	mspID       = "Org1MSP"
	gatewayPeer = "peer0.org1.example.com"
)

// Gateway holds the Fabric network connection and contract reference.
type Gateway struct {
	Contract *client.Contract
	gw       *client.Gateway
	conn     *grpc.ClientConn
}

// NewGateway initializes a connection to the Hyperledger Fabric network.
func NewGateway(cfg *config.Config) (*Gateway, error) {
	certPath := path.Join(cfg.CryptoPath, "users/User1@org1.example.com/msp/signcerts")
	keyPath := path.Join(cfg.CryptoPath, "users/User1@org1.example.com/msp/keystore")
	tlsCertPath := path.Join(cfg.CryptoPath, "peers/peer0.org1.example.com/tls/ca.crt")

	// Peer address: env var or default localhost
	peerAddr := os.Getenv("FABRIC_PEER_ADDRESS")
	if peerAddr == "" {
		peerAddr = "dns:///localhost:7051"
	}

	// gRPC connection
	certPEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		return nil, fmt.Errorf("read TLS cert: %w", err)
	}
	cert, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	certPool.AddCert(cert)
	creds := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	conn, err := grpc.NewClient(peerAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("gRPC dial: %w", err)
	}

	// Identity
	certPEM, err = readFirstFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read identity cert: %w", err)
	}
	x509Cert, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		return nil, err
	}
	id, err := identity.NewX509Identity(mspID, x509Cert)
	if err != nil {
		return nil, err
	}

	// Signer
	keyPEM, err := readFirstFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	privateKey, err := identity.PrivateKeyFromPEM(keyPEM)
	if err != nil {
		return nil, err
	}
	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, err
	}

	// Gateway
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(conn),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("gateway connect: %w", err)
	}

	chaincode := os.Getenv("CHAINCODE_NAME")
	if chaincode == "" {
		chaincode = "basic"
	}
	channel := os.Getenv("CHANNEL_NAME")
	if channel == "" {
		channel = "mychannel"
	}

	network := gw.GetNetwork(channel)
	contract := network.GetContract(chaincode)

	return &Gateway{
		Contract: contract,
		gw:       gw,
		conn:     conn,
	}, nil
}

// Close releases the Fabric gateway and gRPC connection.
func (g *Gateway) Close() {
	if g.gw != nil {
		g.gw.Close()
	}
	if g.conn != nil {
		g.conn.Close()
	}
}

func readFirstFile(dirPath string) ([]byte, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(1)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path.Join(dirPath, names[0]))
}
