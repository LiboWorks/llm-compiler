// C++ wrapper - same implementation as the previous C file but compiled as C++
#include "llama_wrapper.h"
#include "llama.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

struct LlamaModelHandle {
    struct llama_model *model;
    struct llama_context *ctx;
};

// helper
static char *strdup_m(const char *s) {
    if (!s) return NULL;
    size_t n = strlen(s);
    char *r = (char*)malloc(n + 1);
    if (r) memcpy(r, s, n + 1);
    return r;
}

LlamaModelHandle *llama_load_model(const char *model_path, int n_threads) {
    if (!model_path) return NULL;

    llama_backend_init();

    struct llama_model_params mparams = llama_model_default_params();
    mparams.n_gpu_layers = 0;
    mparams.use_mmap = true;
    mparams.use_mlock = false;

    struct llama_model *model = llama_model_load_from_file(model_path, mparams);
    if (!model) {
        fprintf(stderr, "Failed to load model: %s\n", model_path);
        return NULL;
    }

    struct llama_context_params cparams = llama_context_default_params();
    cparams.n_threads = n_threads;
    cparams.n_threads_batch = n_threads;
    cparams.n_ctx = 2048;

    struct llama_context *ctx = llama_init_from_model(model, cparams);
    if (!ctx) {
        fprintf(stderr, "Failed to create llama context\n");
        llama_model_free(model);
        return NULL;
    }

    LlamaModelHandle *h = (LlamaModelHandle*)malloc(sizeof(LlamaModelHandle));
    h->model = model;
    h->ctx = ctx;
    return h;
}

char *llama_predict(LlamaModelHandle *h, const char *prompt,
                    int max_tokens, float temp, int top_k, float top_p) {
    if (!h || !prompt) return NULL;

    const struct llama_vocab *vocab = llama_model_get_vocab(h->model);

    // Ensure the context's KV cache is fresh for this prediction. Some usage
    // patterns expect each predict to start from an empty KV cache; if the
    // context previously had tokens, feeding a new prompt with positions
    // starting at 0 will cause sequence position mismatches. Resetting the
    // context here avoids that class of errors. This is a conservative
    // approach; callers that need persistent context should maintain their
    // own llama context management.
    if (h->ctx) {
        llama_free(h->ctx);
        struct llama_context_params cparams = llama_context_default_params();
        cparams.n_threads = 4;
        cparams.n_threads_batch = 4;
        cparams.n_ctx = 2048;
        h->ctx = llama_init_from_model(h->model, cparams);
        if (!h->ctx) {
            fprintf(stderr, "Failed to reinit llama context\n");
            return NULL;
        }
    }

    // tokenize prompt
    const int32_t max_prompt_tokens = 1024;
    llama_token tokens[max_prompt_tokens];
    int32_t n_tokens = llama_tokenize(vocab, prompt, strlen(prompt),
                                      tokens, max_prompt_tokens, true, false);
    if (n_tokens < 0) n_tokens = -n_tokens;
    if (n_tokens <= 0) return strdup_m("");

    struct llama_batch batch = llama_batch_init(512, 0, 1);

    // feed prompt
    for (int i = 0; i < n_tokens; i++) {
        batch.token[i] = tokens[i];
        batch.pos[i] = i;
        batch.n_seq_id[i] = 1;
        batch.seq_id[i][0] = 0;
        batch.logits[i] = (i == n_tokens - 1);
    }
    batch.n_tokens = n_tokens;
    llama_decode(h->ctx, batch);

    llama_batch_free(batch);


    struct llama_sampler *smpl = llama_sampler_chain_init(llama_sampler_chain_default_params());
    llama_sampler_chain_add(smpl, llama_sampler_init_top_k(top_k));
    llama_sampler_chain_add(smpl, llama_sampler_init_top_p(top_p, 1));
    llama_sampler_chain_add(smpl, llama_sampler_init_temp(temp));
    llama_sampler_chain_add(smpl, llama_sampler_init_dist(LLAMA_DEFAULT_SEED));

    char *output = (char*)malloc(8192);
    size_t out_pos = 0;

    for (int t = 0; t < max_tokens; t++) {
        llama_token id = llama_sampler_sample(smpl, h->ctx, -1);
        if (llama_vocab_is_eog(vocab, id)) break;

        char piece[256];
        int len = llama_token_to_piece(vocab, id, piece, sizeof(piece), 0, true);
        if (len > 0 && out_pos + len < 8191) {
            memcpy(output + out_pos, piece, len);
            out_pos += len;
        }
        output[out_pos] = '\0';

    struct llama_batch b1 = llama_batch_get_one(&id, 1);
    llama_decode(h->ctx, b1);
    // llama_batch_get_one returns a non-owning batch (it points to stack memory);
    // do NOT call llama_batch_free on it because that would attempt to free
    // memory that was not allocated by malloc and cause a crash.
    }

    llama_sampler_free(smpl);
    output[out_pos] = '\0';
    return output;
}

char *llama_predict_stream(
    LlamaModelHandle *h,
    const char *prompt,
    int max_tokens,
    float temp,
    int top_k,
    float top_p,
    llama_stream_callback on_token,
    void *user_data
) {
    if (!h || !prompt) return NULL;

    const struct llama_vocab *vocab = llama_model_get_vocab(h->model);

    // 1. Tokenize prompt
    llama_token tokens[1024];
    int n_tokens = llama_tokenize(vocab, prompt, strlen(prompt), tokens, 1024, true, false);
    if (n_tokens < 0) n_tokens = -n_tokens;
    if (n_tokens <= 0) return strdup("");

    // 2. Feed prompt into model
    struct llama_batch batch = llama_batch_init(512, 0, 1);
    for (int i = 0; i < n_tokens; i++) {
        batch.token[i] = tokens[i];
        batch.pos[i] = i;
        batch.n_seq_id[i] = 1;
        batch.seq_id[i][0] = 0;
        batch.logits[i] = (i == n_tokens - 1);
    }
    batch.n_tokens = n_tokens;
    llama_decode(h->ctx, batch);
    llama_batch_free(batch);

    // 3. Sampler setup
    struct llama_sampler *smpl = llama_sampler_chain_init(llama_sampler_chain_default_params());
    llama_sampler_chain_add(smpl, llama_sampler_init_top_k(top_k));
    llama_sampler_chain_add(smpl, llama_sampler_init_top_p(top_p, 1));
    llama_sampler_chain_add(smpl, llama_sampler_init_temp(temp));
    llama_sampler_chain_add(smpl, llama_sampler_init_dist(LLAMA_DEFAULT_SEED));

    // 4. Streaming loop
    char *output = (char*)malloc(8192);
    size_t out_pos = 0;

    for (int t = 0; t < max_tokens; t++) {
        llama_token id = llama_sampler_sample(smpl, h->ctx, -1);
        if (llama_vocab_is_eog(vocab, id)) break;

        char piece[256];
        int len = llama_token_to_piece(vocab, id, piece, sizeof(piece), 0, true);

        if (len > 0) {
            // Send piece to callback immediately
            if (on_token) on_token(piece, user_data);

            // Also append to accumulated output
            if (out_pos + len < 8191) {
                memcpy(output + out_pos, piece, len);
                out_pos += len;
                output[out_pos] = '\0';
            }
        }

    struct llama_batch b1 = llama_batch_get_one(&id, 1);
    llama_decode(h->ctx, b1);
    // see comment above: do not free b1 because it's non-owning
    }

    llama_sampler_free(smpl);
    output[out_pos] = '\0';
    return output;
}


void llama_free_string(char *s) {
    if (s) free(s);
}

void llama_close_model(LlamaModelHandle *h) {
    if (!h) return;
    llama_free(h->ctx);
    llama_model_free(h->model);
    llama_backend_free();
    free(h);
}

void llama_reset_context(LlamaModelHandle* h) {
    if (!h) return;
    if (h->ctx) {
        llama_free(h->ctx);
    }
    struct llama_context_params cparams = llama_context_default_params();
    cparams.n_threads = 4;
    cparams.n_threads_batch = 4;
    cparams.n_ctx = 2048;
    h->ctx = llama_init_from_model(h->model, cparams);
}
