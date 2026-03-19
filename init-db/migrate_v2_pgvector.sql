CREATE EXTENSION IF NOT EXISTS vector;

-- 迁移 knowledge_base 表（仅当 embedding 列为 jsonb 类型时执行）
DO $$
BEGIN
    IF (SELECT data_type FROM information_schema.columns
        WHERE table_name = 'knowledge_base' AND column_name = 'embedding') = 'jsonb' THEN

        ALTER TABLE knowledge_base ADD COLUMN embedding_vec vector(1536);
        UPDATE knowledge_base
        SET embedding_vec = (
            '[' || array_to_string(ARRAY(SELECT jsonb_array_elements_text(embedding)), ',') || ']'
        )::vector;
        ALTER TABLE knowledge_base DROP COLUMN embedding;
        ALTER TABLE knowledge_base RENAME COLUMN embedding_vec TO embedding;
        CREATE INDEX IF NOT EXISTS idx_knowledge_base_embedding_hnsw
            ON knowledge_base USING hnsw (embedding vector_cosine_ops)
            WITH (m = 16, ef_construction = 64);

        RAISE NOTICE 'knowledge_base: migrated embedding jsonb -> vector(1536)';
    ELSE
        RAISE NOTICE 'knowledge_base: embedding already vector type, skipping migration';
    END IF;
END $$;

-- 迁移 vector_store 表（仅当 embedding 列为 jsonb 类型时执行）
DO $$
BEGIN
    IF (SELECT data_type FROM information_schema.columns
        WHERE table_name = 'vector_store' AND column_name = 'embedding') = 'jsonb' THEN

        ALTER TABLE vector_store ADD COLUMN embedding_vec vector(1536);
        UPDATE vector_store
        SET embedding_vec = (
            '[' || array_to_string(ARRAY(SELECT jsonb_array_elements_text(embedding)), ',') || ']'
        )::vector;
        ALTER TABLE vector_store DROP COLUMN embedding;
        ALTER TABLE vector_store RENAME COLUMN embedding_vec TO embedding;
        CREATE INDEX IF NOT EXISTS idx_vector_store_embedding_hnsw
            ON vector_store USING hnsw (embedding vector_cosine_ops)
            WITH (m = 16, ef_construction = 64);

        RAISE NOTICE 'vector_store: migrated embedding jsonb -> vector(1536)';
    ELSE
        RAISE NOTICE 'vector_store: embedding already vector type, skipping migration';
    END IF;
END $$;
