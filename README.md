# SwarmStore

A peer-to-peer distributed file storage system built in Go that enables nodes to store, retrieve, and share files across a decentralized network with end-to-end encryption.

## Overview

This project implements a distributed file storage system where multiple nodes communicate over TCP to share files. Each node maintains its own local storage and can request files from or serve files to other nodes in the network. The system is designed to work in a fully decentralized manner without requiring a central server.

## Features

- **Peer-to-Peer File Sharing**: Nodes can directly exchange files without a central server. Any node can serve files to any other node on the network.
- **End-to-End Encryption**: All file data is encrypted using AES-128 in CTR mode before network transmission, ensuring data privacy during transit.
- **Content Addressable Storage**: Files are stored using hashed paths (CAS style), which provides natural file deduplication and efficient storage management.
- **Distributed Operations**: File deletion requests propagate across the network, ensuring consistency across all nodes.
- **Multi-Peer Broadcasting**: Files can be broadcast to multiple peers simultaneously using Go's io.MultiWriter.
- **Bootstrapping**: New nodes can join the network by connecting to bootstrap nodes, enabling organic network growth.
- **Connection Handling**: Supports both inbound and outbound peer connections with proper stream management.

## How It Works

### Network Communication

The system uses TCP for all peer-to-peer communication. When a node starts, it listens for incoming connections and can optionally connect to bootstrap nodes to discover the network. Each node maintains a map of connected peers and can broadcast messages or send direct requests.

### File Storage

Files are stored locally on each node using a path transformation function. The default implementation generates a CAS-style path by hashing the file key and splitting the hash into segments to create a directory structure. This approach prevents having too many files in a single directory and enables efficient lookups.

### Encryption

Before transmitting file data over the network, the content is encrypted using AES-128 in CTR mode. The encryption key is unique to each node. The initialization vector (IV) is randomly generated for each encryption operation and prepended to the encrypted data so that the receiving node can decrypt it.

### Message Protocol

The system uses a simple binary protocol:

- Regular messages are encoded using Go's `gob` encoding and sent with a message flag
- File transfers use a stream flag, followed by the file size and encrypted content
- Three message types handle the core operations: store, get, and delete

## Try It On Your System

```bash
git clone https://github.com/ShivankSharma070/SwarmStore
cd SwarmStore
make run
```

The main function creates a three-node network. Explore the code to understand how to build your own use cases.

## Key Concepts

### Nodes and Peers

Each running instance of the file server is a node. Connected instances are referred to as peers. The system distinguishes between outbound connections (dialed) and inbound connections (accepted).

### Path Transformation

The path transformation function converts a file key into a filesystem path. This allows flexible storage strategies. The default CAS implementation splits a SHA1 hash into segments to create nested directories.

### Stream Management

When transferring files, the system uses stream handling to prevent concurrent reads on the same connection. Peers signal the start and end of streams, allowing the receiver to know when the transfer is complete.

## Requirements

- Go 1.21 or later

## Security Considerations

- Each node should maintain its own encryption key
- The current implementation does not include authentication between peers
- Network connections are unencrypted (only file content is encrypted)

## Future Improvements

- Add peer authentication and secure handshakes
- Implement file synchronization across nodes
- Add chunking for large files
- Support for additional transport protocols
- Implement a discovery mechanism for finding peers

## License

MIT
