# 🌐 libp2p-learn

A comprehensive learning project exploring **libp2p** (peer-to-peer networking library) implementations across different programming languages. This repository serves as both a learning resource and a practical implementation showcase for building distributed, decentralized applications.

## 📋 **Project Overview**

libp2p is a modular system of protocols, specifications, and libraries that enable the development of peer-to-peer network applications. This project demonstrates:

- **P2P Node Creation**: Building singular nodes that can join peer-to-peer networks
- **Multi-transport Support**: TCP, UDP (QUIC), and WebSocket connectivity
- **Advanced Features**: Hole punching, NAT traversal, DHT routing, relay services
- **Scalability**: Handling many simultaneous connections
- **Real-world Protocols**: Custom application-level protocols (Ping, Chat, Echo)

---

## 🗂️ **Repository Structure**

```
libp2p-learn/
├── README.md                 # This overview document
├── libp2p-go/               # Go implementation (Complete)
│   ├── README.md            # Detailed Go-specific documentation
│   ├── main.go              # CLI application entry point
│   ├── node.go              # Core libp2p node implementation
│   ├── protocols.go         # Custom protocol handlers
│   ├── config.go            # Configuration management
│   ├── bootstrap.go         # Network bootstrapping
│   ├── dht_test.go          # DHT functionality tests
│   ├── integration_test.go  # Multi-node integration tests
│   ├── node_test.go         # Unit tests for node creation
│   ├── test_helpers.go      # Test synchronization utilities
│   ├── Makefile             # Build automation
│   ├── Dockerfile           # Container configuration
│   ├── go.mod               # Go dependency management
│   └── ...
└── libp2p-rust/             # Rust implementation (Planned)
    └── README.md            # Rust-specific documentation
```

---

## 🚀 **Quick Start**

### **Go Implementation** (Ready to Use)

The Go implementation is **complete and fully functional**. Navigate to the `libp2p-go/` directory for detailed setup instructions.

```bash
cd libp2p-go/
make build
make run-example
```

**Key Features:**
- ✅ TCP, UDP (QUIC), WebSocket transports
- ✅ DHT-based peer discovery and data storage
- ✅ Circuit relay for NAT traversal
- ✅ Hole punching with DCUtR protocol
- ✅ AutoNAT for automatic NAT detection
- ✅ Comprehensive test suite with deterministic behavior
- ✅ Docker support for containerized deployment
- ✅ CLI interface with Cobra
- ✅ Structured logging with Logrus

### **Rust Implementation** (Coming Soon)

The Rust implementation is planned and will provide similar functionality using the Rust libp2p ecosystem.

```bash
cd libp2p-rust/
# Instructions coming soon...
```

---

## 🎯 **Learning Objectives**

This project helps you understand:

1. **P2P Networking Fundamentals**
   - Peer discovery and connection management
   - Distributed hash tables (DHT)
   - Network topologies and routing

2. **libp2p Ecosystem**
   - Multiaddress format and transport abstraction
   - Stream multiplexing and protocol negotiation
   - Security layers (Noise, TLS)

3. **Advanced Networking Concepts**
   - NAT traversal techniques
   - Hole punching protocols
   - Circuit relay patterns

4. **Distributed Systems**
   - Gossip protocols
   - Consensus mechanisms
   - Network resilience

5. **Implementation Patterns**
   - Event-driven architecture
   - Asynchronous programming
   - Testing distributed systems

---

## 🛠️ **Development**

### **Prerequisites**

- **Go**: Version 1.23+ (for Go implementation)
- **Rust**: Version 1.70+ (for future Rust implementation)
- **Docker**: For containerized testing
- **Make**: For build automation

### **Repository Setup**

```bash
git clone <repository-url>
cd libp2p-learn

# For Go development
cd libp2p-go
make dev-tools
make test

# For Rust development (future)
cd libp2p-rust
# Setup instructions coming soon...
```

---

## 🧪 **Testing Philosophy**

Both implementations emphasize **deterministic testing**:

- **Event-driven synchronization** instead of arbitrary time delays
- **Channel-based coordination** for reliable test execution
- **Comprehensive integration tests** covering real-world scenarios
- **Race condition detection** for concurrent operations

---

## 📚 **Resources**

### **libp2p Documentation**
- [libp2p Specification](https://github.com/libp2p/specs)
- [go-libp2p Documentation](https://docs.libp2p.io/)
- [rust-libp2p Documentation](https://docs.rs/libp2p/)

### **Networking Concepts**
- [Kademlia DHT Paper](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf)
- [NAT Traversal Techniques](https://datatracker.ietf.org/doc/html/rfc5389)
- [WebRTC and STUN/TURN](https://webrtcforthecurious.com/)

---

## 🎓 **Educational Note**

This is a learning project designed to explore libp2p concepts and implementations. While the code is production-quality, it's primarily intended for educational purposes and understanding distributed systems concepts.

**Happy Learning! 🚀**

---

*For specific implementation details, navigate to the respective language directories (`libp2p-go/` or `libp2p-rust/`) and check their individual README files.* 