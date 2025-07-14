# xSync Tweet Embedding System

## Overview

The xSync Tweet Embedding System provides semantic search capabilities for tweets using ChromaDB and SentenceTransformers. This system enables LLM-profitable web3 token analysis by storing tweet embeddings and performing similarity-based searches.

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   SQLite DB     │    │  Embedding API  │    │   ChromaDB      │
│   (Tweets)      │◄───┤  (Python)       │────┤  (Vectors)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         │                        │                        │
         └────────────────────────┼────────────────────────┘
                                  │
                      ┌─────────────────┐
                      │   Embedder CLI  │
                      │   (Go)          │
                      └─────────────────┘
```

## Components

### 1. ChromaDB (Vector Database)
- **Purpose**: Stores tweet embeddings and metadata
- **Port**: 8000
- **Authentication**: Token-based (xsync-dev-token-2025)

### 2. Embedding API (Python Service)
- **Purpose**: Generates embeddings using SentenceTransformers
- **Port**: 8001
- **Models**: 
  - Embedding: `sentence-transformers/all-MiniLM-L6-v2`
  - Re-ranking: `cross-encoder/ms-marco-MiniLM-L-6-v2`

### 3. Redis (Caching)
- **Purpose**: Caches embeddings and search results
- **Port**: 6379
- **Password**: xsync-redis-2025

### 4. Embedder CLI (Go Application)
- **Purpose**: Manages tweet indexing and searching
- **Features**: Batch indexing, semantic search, web3 analysis

## Quick Start

### 1. Launch Infrastructure

```bash
# Make script executable
chmod +x scripts/launch-chromadb.sh

# Start all services (ChromaDB, Redis, Embedding API)
./scripts/launch-chromadb.sh start

# Check service status
./scripts/launch-chromadb.sh status

# View service endpoints
./scripts/launch-chromadb.sh endpoints
```

### 2. Build and Use Embedder CLI

```bash
# Build the embedder CLI
go build -o bin/embedder ./cmd/embedder

# Index all tweets from database
./bin/embedder -cmd=index-all

# Search for web3-related tweets
./bin/embedder -cmd=search-web3 -query="DeFi yield farming"

# Get indexing statistics
./bin/embedder -cmd=stats

# Start auto-indexer (continuous indexing)
./bin/embedder -cmd=auto -interval=10m
```

## Features

### Web3 Token Analysis
- **Token Detection**: Automatically detects mentions of cryptocurrencies, DeFi protocols, NFTs
- **Sentiment Analysis**: Basic sentiment scoring for token-related content
- **Categorization**: Classifies content into DeFi, NFT, Gaming, etc.

### Semantic Search
- **Similarity Search**: Find tweets similar to a query using embeddings
- **Filtering**: Filter by user, date range, web3 content, sentiment
- **Re-ranking**: Uses CrossEncoder for improved result relevance

### Batch Processing
- **Efficient Indexing**: Processes tweets in configurable batches
- **Error Handling**: Continues processing despite individual failures
- **Progress Tracking**: Logs processing progress and statistics

## Configuration

### Environment Variables

```bash
# ChromaDB Configuration
CHROMA_URL=http://localhost:8000
CHROMA_TOKEN=xsync-dev-token-2025

# Redis Configuration  
REDIS_URL=localhost:6379
REDIS_PASSWORD=xsync-redis-2025

# Embedding API Configuration
EMBEDDING_API_URL=http://localhost:8001
EMBEDDING_MODEL=sentence-transformers/all-MiniLM-L6-v2
```

### CLI Options

```bash
# Database and service configuration
-db string          SQLite database path (default: ./conf/data/xSync.db)
-chroma-url string  ChromaDB URL (default: http://localhost:8000)
-chroma-token string ChromaDB token (default: xsync-dev-token-2025)
-redis-url string   Redis URL (default: localhost:6379)
-redis-pass string  Redis password (default: xsync-redis-2025)

# Processing options
-limit int          Search result limit (default: 50)
-interval duration  Auto-indexing interval (default: 5m)
-user string        User ID for user-specific operations
-query string       Search query
```

## Commands

### Indexing Commands

```bash
# Index all tweets from database
./bin/embedder -cmd=index-all

# Index new tweets (last 24 hours)
./bin/embedder -cmd=index-new

# Index tweets for specific user
./bin/embedder -cmd=index-user -user=123456789

# Start continuous auto-indexing
./bin/embedder -cmd=auto -interval=10m
```

### Search Commands

```bash
# General semantic search
./bin/embedder -cmd=search -query="blockchain technology" -limit=20

# Web3-specific search
./bin/embedder -cmd=search-web3 -query="DeFi protocols" -limit=10

# Get indexing statistics
./bin/embedder -cmd=stats
```

## API Endpoints

### Embedding API (Port 8001)

```bash
# Health check
curl http://localhost:8001/health

# Generate single embedding
curl -X POST http://localhost:8001/embed \
  -H "Content-Type: application/json" \
  -d '{"text": "Bitcoin is the future of money"}'

# Generate batch embeddings
curl -X POST http://localhost:8001/embed/batch \
  -H "Content-Type: application/json" \
  -d '{"texts": ["Bitcoin price", "Ethereum staking"]}'

# Re-rank documents
curl -X POST http://localhost:8001/rerank \
  -H "Content-Type: application/json" \
  -d '{"query": "DeFi", "documents": ["Uniswap AMM", "Bitcoin mining"]}'
```

### ChromaDB API (Port 8000)

```bash
# Health check
curl -H "X-Chroma-Token: xsync-dev-token-2025" \
  http://localhost:8000/api/v1/heartbeat

# List collections
curl -H "X-Chroma-Token: xsync-dev-token-2025" \
  http://localhost:8000/api/v1/collections
```

## Web3 Token Analysis

### Supported Token Categories
- **DeFi**: Uniswap, Aave, Compound, Curve, etc.
- **Layer 1**: Bitcoin, Ethereum, Solana, Polkadot, etc.
- **Layer 2**: Polygon, Arbitrum, Optimism, etc.
- **NFT**: OpenSea, CryptoPunks, BAYC, etc.
- **Gaming**: Axie Infinity, Decentraland, The Sandbox, etc.

### Sentiment Analysis
- **Positive**: bullish, moon, pump, buy, good, great
- **Negative**: bearish, dump, crash, sell, scam, bad
- **Neutral**: Default when no strong sentiment indicators

### Example Output

```json
{
  "tweet": {
    "id": "tweet_1234567890",
    "content": "Just bought more $ETH for DeFi farming. Bullish on yield opportunities!",
    "user_name": "CryptoTrader",
    "metadata": {
      "is_web3": true,
      "web3_tokens": ["eth", "defi", "yield", "farming", "bullish"],
      "sentiment": "positive",
      "sentiment_score": 0.8,
      "web3_score": 0.9
    }
  },
  "score": 0.95,
  "rank": 1
}
```

## Monitoring and Maintenance

### Service Health Checks

```bash
# Check all services
./scripts/launch-chromadb.sh status

# View logs
./scripts/launch-chromadb.sh logs
./scripts/launch-chromadb.sh logs chromadb
./scripts/launch-chromadb.sh logs embedding-api
```

### Performance Monitoring

```bash
# View indexing statistics
./bin/embedder -cmd=stats

# Monitor Redis cache
redis-cli -h localhost -p 6379 -a xsync-redis-2025 info memory

# Monitor ChromaDB collection size
curl -H "X-Chroma-Token: xsync-dev-token-2025" \
  http://localhost:8000/api/v1/collections/xsync_tweets
```

## Troubleshooting

### Common Issues

1. **Service not starting**: Check Docker and ensure ports are available
2. **Embedding API timeout**: Models take time to load on first start
3. **ChromaDB connection failed**: Verify token and URL configuration
4. **Out of memory**: Reduce batch size or increase Docker memory limits

### Error Recovery

```bash
# Restart all services
./scripts/launch-chromadb.sh restart

# Clear Redis cache
redis-cli -h localhost -p 6379 -a xsync-redis-2025 flushdb

# Re-index with smaller batches
./bin/embedder -cmd=index-all -batch-size=50
```

## Development

### Adding New Features

1. **Custom Token Detection**: Update `analyzeWeb3Content()` in `embedder.go`
2. **New Search Filters**: Extend `SearchQuery` struct in `types.go`
3. **Additional Models**: Update embedding API configuration
4. **Custom Metrics**: Extend `IndexStats` structure

### Testing

```bash
# Test embedding API
curl http://localhost:8001/health

# Test simple search
./bin/embedder -cmd=search -query="test" -limit=5

# Verify database schema
sqlite3 ./conf/data/xSync.db ".schema tweet_embeddings"
```

## Future Enhancements

- **Real-time Indexing**: WebSocket integration for live tweet processing
- **Advanced Analytics**: Price correlation analysis and prediction
- **Multi-language Support**: Non-English tweet processing
- **Graph Embeddings**: User and token relationship modeling
- **ML Pipeline**: Automated model training and updating
