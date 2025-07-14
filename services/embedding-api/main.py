import os
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional
import numpy as np
from sentence_transformers import SentenceTransformer, CrossEncoder
import logging
import time
from contextlib import asynccontextmanager

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Global models (loaded at startup)
embedding_model = None
cross_encoder = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Load models at startup
    global embedding_model, cross_encoder
    
    logger.info("Loading embedding models...")
    start_time = time.time()
    
    # Load SentenceTransformer model
    model_name = os.getenv("EMBEDDING_MODEL", "sentence-transformers/all-MiniLM-L6-v2")
    embedding_model = SentenceTransformer(model_name)
    logger.info(f"Loaded embedding model: {model_name}")
    
    # Load CrossEncoder for re-ranking
    cross_encoder_name = os.getenv("CROSS_ENCODER_MODEL", "cross-encoder/ms-marco-MiniLM-L-6-v2")
    cross_encoder = CrossEncoder(cross_encoder_name)
    logger.info(f"Loaded cross-encoder model: {cross_encoder_name}")
    
    load_time = time.time() - start_time
    logger.info(f"Models loaded in {load_time:.2f} seconds")
    
    yield
    
    # Cleanup (if needed)
    logger.info("Shutting down embedding service")

# Create FastAPI app with lifespan
app = FastAPI(
    title="xSync Embedding API",
    description="Sentence Transformer and Cross-Encoder API for xSync tweet embeddings",
    version="1.0.0",
    lifespan=lifespan
)

class EmbeddingRequest(BaseModel):
    text: str
    model: Optional[str] = None

class BatchEmbeddingRequest(BaseModel):
    texts: List[str]
    model: Optional[str] = None

class EmbeddingResponse(BaseModel):
    embedding: List[float]
    model: str
    dimension: int
    processing_time: float

class BatchEmbeddingResponse(BaseModel):
    embeddings: List[List[float]]
    model: str
    dimension: int
    processing_time: float
    count: int

class RerankRequest(BaseModel):
    query: str
    documents: List[str]
    top_k: Optional[int] = 10

class RerankResponse(BaseModel):
    scores: List[float]
    rankings: List[int]
    processing_time: float

class HealthResponse(BaseModel):
    status: str
    model_loaded: bool
    cross_encoder_loaded: bool
    uptime: float

# Store startup time for uptime calculation
startup_time = time.time()

@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    return HealthResponse(
        status="healthy",
        model_loaded=embedding_model is not None,
        cross_encoder_loaded=cross_encoder is not None,
        uptime=time.time() - startup_time
    )

@app.post("/embed", response_model=EmbeddingResponse)
async def create_embedding(request: EmbeddingRequest):
    """Create embedding for a single text"""
    if embedding_model is None:
        raise HTTPException(status_code=503, detail="Embedding model not loaded")
    
    if not request.text.strip():
        raise HTTPException(status_code=400, detail="Text cannot be empty")
    
    try:
        start_time = time.time()
        
        # Generate embedding
        embedding = embedding_model.encode(request.text, convert_to_tensor=False)
        
        # Convert to list and ensure float32
        embedding_list = embedding.astype(np.float32).tolist()
        
        processing_time = time.time() - start_time
        
        return EmbeddingResponse(
            embedding=embedding_list,
            model=embedding_model.get_model_card()["model_name"] if hasattr(embedding_model, 'get_model_card') else "unknown",
            dimension=len(embedding_list),
            processing_time=processing_time
        )
        
    except Exception as e:
        logger.error(f"Error generating embedding: {e}")
        raise HTTPException(status_code=500, detail=f"Error generating embedding: {str(e)}")

@app.post("/embed/batch", response_model=BatchEmbeddingResponse)
async def create_batch_embeddings(request: BatchEmbeddingRequest):
    """Create embeddings for multiple texts"""
    if embedding_model is None:
        raise HTTPException(status_code=503, detail="Embedding model not loaded")
    
    if not request.texts or len(request.texts) == 0:
        raise HTTPException(status_code=400, detail="Texts list cannot be empty")
    
    # Filter out empty texts
    valid_texts = [text for text in request.texts if text.strip()]
    if len(valid_texts) == 0:
        raise HTTPException(status_code=400, detail="No valid texts provided")
    
    try:
        start_time = time.time()
        
        # Generate embeddings in batch
        embeddings = embedding_model.encode(valid_texts, convert_to_tensor=False, batch_size=32)
        
        # Convert to list and ensure float32
        embeddings_list = [emb.astype(np.float32).tolist() for emb in embeddings]
        
        processing_time = time.time() - start_time
        
        return BatchEmbeddingResponse(
            embeddings=embeddings_list,
            model=embedding_model.get_model_card()["model_name"] if hasattr(embedding_model, 'get_model_card') else "unknown",
            dimension=len(embeddings_list[0]) if embeddings_list else 0,
            processing_time=processing_time,
            count=len(embeddings_list)
        )
        
    except Exception as e:
        logger.error(f"Error generating batch embeddings: {e}")
        raise HTTPException(status_code=500, detail=f"Error generating batch embeddings: {str(e)}")

@app.post("/rerank", response_model=RerankResponse)
async def rerank_documents(request: RerankRequest):
    """Re-rank documents using CrossEncoder"""
    if cross_encoder is None:
        raise HTTPException(status_code=503, detail="Cross-encoder model not loaded")
    
    if not request.query.strip():
        raise HTTPException(status_code=400, detail="Query cannot be empty")
    
    if not request.documents or len(request.documents) == 0:
        raise HTTPException(status_code=400, detail="Documents list cannot be empty")
    
    try:
        start_time = time.time()
        
        # Create query-document pairs
        pairs = [[request.query, doc] for doc in request.documents]
        
        # Get relevance scores
        scores = cross_encoder.predict(pairs)
        
        # Convert to list and ensure float32
        scores_list = scores.astype(np.float32).tolist()
        
        # Create rankings (indices sorted by score descending)
        rankings = sorted(range(len(scores_list)), key=lambda i: scores_list[i], reverse=True)
        
        # Limit to top_k if specified
        if request.top_k and request.top_k < len(rankings):
            rankings = rankings[:request.top_k]
        
        processing_time = time.time() - start_time
        
        return RerankResponse(
            scores=scores_list,
            rankings=rankings,
            processing_time=processing_time
        )
        
    except Exception as e:
        logger.error(f"Error re-ranking documents: {e}")
        raise HTTPException(status_code=500, detail=f"Error re-ranking documents: {str(e)}")

@app.get("/models/info")
async def get_model_info():
    """Get information about loaded models"""
    return {
        "embedding_model": {
            "loaded": embedding_model is not None,
            "name": os.getenv("EMBEDDING_MODEL", "sentence-transformers/all-MiniLM-L6-v2"),
            "dimension": embedding_model.get_sentence_embedding_dimension() if embedding_model else None
        },
        "cross_encoder": {
            "loaded": cross_encoder is not None,
            "name": os.getenv("CROSS_ENCODER_MODEL", "cross-encoder/ms-marco-MiniLM-L-6-v2")
        }
    }

if __name__ == "__main__":
    import uvicorn
    
    # Configuration from environment
    host = os.getenv("HOST", "0.0.0.0")
    port = int(os.getenv("PORT", "8001"))
    workers = int(os.getenv("WORKERS", "1"))
    
    # Run the server
    uvicorn.run(
        "main:app",
        host=host,
        port=port,
        workers=workers,
        log_level="info",
        reload=False
    )
