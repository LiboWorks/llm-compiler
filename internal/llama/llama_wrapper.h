#ifndef LLAMA_WRAPPER_H
#define LLAMA_WRAPPER_H

#include <stdint.h>

// Ensure C linkage when included from C++ sources
#ifdef __cplusplus
extern "C" {
#endif

// Opaque handle to a loaded model/context
typedef struct LlamaModelHandle LlamaModelHandle;

typedef void (*llama_stream_callback)(const char *token_text, void *user_data);


// Load a model from a file path and return a handle, or NULL on error.
// Caller takes ownership and must call llama_close_model(handle).
LlamaModelHandle* llama_load_model(const char* model_path, int n_threads);

// Run prediction for a prompt. Returns a malloc'd C string (caller must free).
// max_tokens: maximum tokens to generate
// returns NULL on error.
char* llama_predict(LlamaModelHandle* h, const char* prompt, int max_tokens, float temp, int top_k, float top_p);

// Reset the context (KV cache) for a loaded model handle. This frees the
// existing context and creates a fresh one so subsequent predictions start
// with an empty KV cache.
void llama_reset_context(LlamaModelHandle* h);

char *llama_predict_stream(
    LlamaModelHandle *h,
    const char *prompt,
    int max_tokens,
    float temp,
    int top_k,
    float top_p,
    llama_stream_callback on_token,
    void *user_data
);


// Free the C string returned by llama_predict
void llama_free_string(char* s);

// Close model and free resources
void llama_close_model(LlamaModelHandle* h);

#ifdef __cplusplus
}
#endif

#endif // LLAMA_WRAPPER_H
