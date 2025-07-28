# RAG (Retrieval-Augmented Generation) Feature for Token Detection

## Overview

This RAG feature adds intelligent token symbol detection to the xSync Twitter media downloader. When new tweets are processed, the system automatically analyzes them for potential cryptocurrency token mentions and queries a ChromaDB vector database to find related token symbols for later LLM analysis.

## Components

### 1. RAG Analyzer (`/pkgs/ragpkg/analyzer/`)

The core RAG analysis functionality:

- **`rag_analyzer.go`**: Main analyzer that processes tweets and queries ChromaDB
- **`tweet_analysis_service.go`**: Service wrapper for async tweet analysis
- **`db_worker_with_rag.go`**: Enhanced database worker with RAG integration

### 2. Entry Points

- **`/cmd/rag-analyzer/main.go`**: Standalone RAG analyzer service
- Enhanced CLI integration via `db_worker_with_rag.go`

### 3. Example Integration (`/examples/`)

- **`rag_integration_example.go`**: Sample code showing how to use the RAG functionality

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   New Tweet     │───▶│  RAG Analyzer   │───▶│   ChromaDB      │
│  Processing     │    │   Service       │    │  Vector Store   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │ Analysis Results │
                       │ (JSON Storage)   │
                       └─────────────────┘
```

## Features

### Token Detection
- Regex-based extraction of potential token symbols ($BTC, $ETH, etc.)
- Hashtag analysis for token mentions (#DeFi, #Ethereum)
- Context-aware filtering to reduce false positives

### ChromaDB Integration
- Vector similarity search for related tokens
- Configurable similarity thresholds
- Batch processing for performance

### Async Processing
- Non-blocking tweet analysis
- Configurable concurrency levels
- Error handling and retry logic

## Setup and Configuration

### Prerequisites

1. **ChromaDB Server**: Running instance of ChromaDB
2. **PostgreSQL Database**: For storing tweets and analysis results
3. **Go Environment**: Go 1.19+ with module support

### ChromaDB Setup

1. Install and run ChromaDB:
```bash
pip install chromadb
chroma run --host localhost --port 8000
```

2. Populate with token data (example):
```python
import chromadb

client = chromadb.HttpClient(host="localhost", port=8000)
collection = client.create_collection("token_symbols")

# Add token symbols with metadata
collection.add(
    documents=["Bitcoin cryptocurrency", "Ethereum blockchain platform"],
    metadatas=[{"symbol": "BTC"}, {"symbol": "ETH"}],
    ids=["btc", "eth"]
)
```

### Configuration

Update your configuration to include ChromaDB settings:

```yaml
# config.yaml
chroma:
  host: "localhost"
  port: 8000
  collection: "token_symbols"
  similarity_threshold: 0.7

rag:
  enabled: true
  async_processing: true
  batch_size: 10
  analysis_interval: "30s"
```

## Usage

### Standalone RAG Analyzer

Build and run the dedicated RAG analyzer:

```bash
go build -o rag-analyzer ./cmd/rag-analyzer
./rag-analyzer
```

### Integrated Mode

The RAG functionality is automatically enabled when using the enhanced database worker:

```bash
# Normal tweet processing with RAG analysis
go run ./cmd/cli --user elonmusk
```

### Programmatic Usage

```go
package main

import (
    "context"
    "github.com/WangWilly/xSync/pkgs/ragpkg/analyzer"
    "github.com/WangWilly/xSync/pkgs/commonpkg/clients/chromatokenclient"
)

func main() {
    // Initialize ChromaDB client
    chromaClient, err := chromatokenclient.New("http://localhost:8000")
    if err != nil {
        panic(err)
    }

    // Create RAG analyzer
    ragAnalyzer := analyzer.NewRAGAnalyzer(db, chromaClient, tweetRepo, tokenRepo)

    // Analyze a tweet
    result, err := ragAnalyzer.AnalyzeTweet(context.Background(), tweet)
    if err != nil {
        // handle error
    }

    // Process results
    for _, token := range result.PotentialTokens {
        fmt.Printf("Found potential token: %s\n", token)
    }
}
```

## API Reference

### RAGAnalyzer

#### Methods

- `AnalyzeTweet(ctx context.Context, tweet *model.Tweet) (*TweetAnalysisResult, error)`
  - Analyzes a single tweet for token mentions
  
- `StartContinuousAnalysis(ctx context.Context) error`
  - Starts continuous background analysis of new tweets

#### TweetAnalysisResult

```go
type TweetAnalysisResult struct {
    TweetID           uint64   `json:"tweet_id"`
    Content           string   `json:"content"`
    PotentialTokens   []string `json:"potential_tokens"`
    ChromaMatches     []string `json:"chroma_matches"`
    ConfidenceScore   float64  `json:"confidence_score"`
    RecommendedTokens []string `json:"recommended_tokens"`
}
```

### TweetAnalysisService

#### Methods

- `AnalyzeNewTweet(ctx context.Context, tweet *model.Tweet)`
  - Asynchronously analyzes a new tweet (fire-and-forget)

## Performance Considerations

### Optimization Settings

```go
// Recommended settings for high-volume processing
analyzer := analyzer.NewRAGAnalyzer(db, chromaClient, tweetRepo, tokenRepo)
service := analyzer.NewTweetAnalysisService(analyzer, true) // Enable async
```

### Monitoring

The system provides detailed logging for monitoring:

```
INFO[2024-01-15T10:30:00Z] Starting continuous tweet analysis for token detection
DEBUG[2024-01-15T10:30:01Z] Analyzing tweet: "Just bought some $BTC..."
DEBUG[2024-01-15T10:30:01Z] Found 2 potential tokens: [BTC, ETH]
DEBUG[2024-01-15T10:30:02Z] ChromaDB query returned 3 similar tokens
```

## Error Handling

The system includes comprehensive error handling:

- ChromaDB connection failures are logged but don't stop tweet processing
- Invalid tweet content is skipped with warnings
- Database errors are retried with exponential backoff

## Testing

Run the test suite:

```bash
go test ./pkgs/ragpkg/analyzer/...
```

## Troubleshooting

### Common Issues

1. **ChromaDB Connection Failed**
   - Verify ChromaDB is running: `curl http://localhost:8000/api/v1/heartbeat`
   - Check firewall settings

2. **No Token Matches Found**
   - Verify ChromaDB collection has data
   - Adjust similarity threshold in configuration

3. **High CPU Usage**
   - Reduce batch size in configuration
   - Increase analysis interval

### Debug Mode

Enable debug logging:

```bash
export LOG_LEVEL=debug
./rag-analyzer
```

## Future Enhancements

- [ ] Integration with OpenAI/LLM APIs for advanced analysis
- [ ] Support for multiple ChromaDB collections
- [ ] Real-time WebSocket API for analysis results
- [ ] Machine learning model training on analysis results
- [ ] Integration with external token price APIs

## Contributing

When contributing to the RAG functionality:

1. Follow the existing code patterns in `/pkgs/ragpkg/`
2. Add comprehensive tests for new features
3. Update this documentation
4. Ensure backward compatibility with existing tweet processing

## License

This RAG feature follows the same license as the main xSync project.
