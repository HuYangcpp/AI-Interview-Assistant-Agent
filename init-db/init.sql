-- 启用必要的扩展
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS cube;
CREATE EXTENSION IF NOT EXISTS earthdistance;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 创建向量存储表
CREATE TABLE IF NOT EXISTS "public"."vector_store" (
   "id" BIGSERIAL PRIMARY KEY,
   "chat_id" varchar(255) NOT NULL,
    "role" varchar(50) NOT NULL,
    "content" TEXT NOT NULL,
    "embedding" vector(1536) NOT NULL,
    "source_type" VARCHAR(50) NOT NULL DEFAULT 'message',
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT now()
    );

-- 创建知识库表
CREATE TABLE IF NOT EXISTS "public"."knowledge_base" (
    "id" BIGSERIAL PRIMARY KEY,
    "title" VARCHAR(255) NOT NULL,
    "content" TEXT NOT NULL,
    "embedding" vector(1536) NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_vector_store_chat_id ON vector_store (chat_id);
CREATE INDEX IF NOT EXISTS idx_vector_store_created_at ON vector_store (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_vector_store_embedding_hnsw
    ON vector_store USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
CREATE INDEX IF NOT EXISTS idx_knowledge_base_title ON knowledge_base (title);
CREATE INDEX IF NOT EXISTS idx_knowledge_base_embedding_hnsw
    ON knowledge_base USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- =========================
-- 学生就业信息管理系统新增业务表
-- 保留 vector_store 与 knowledge_base 不变
-- =========================

-- users（用户表）
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      VARCHAR(50)  NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(20)  NOT NULL DEFAULT 'student', -- admin | student
    real_name     VARCHAR(50)  NOT NULL DEFAULT '',
    phone         VARCHAR(20)  DEFAULT '',
    email         VARCHAR(100) DEFAULT '',
    status        SMALLINT     NOT NULL DEFAULT 1, -- 1=启用 0=禁用
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- students（学生信息表）
CREATE TABLE IF NOT EXISTS students (
    id                BIGSERIAL PRIMARY KEY,
    user_id           BIGINT      NOT NULL UNIQUE REFERENCES users(id),
    student_no        VARCHAR(30) NOT NULL UNIQUE,  -- 学号
    major             VARCHAR(100) DEFAULT '',      -- 专业
    class_name        VARCHAR(50)  DEFAULT '',      -- 班级
    graduation_year   INT          DEFAULT 0,       -- 毕业年份
    skills            TEXT         DEFAULT '[]',    -- JSON 数组
    self_introduction TEXT         DEFAULT '',
    resume_url        VARCHAR(500) DEFAULT '',
    resume_mode       VARCHAR(20)  NOT NULL DEFAULT 'short',
    resume_length     INT          NOT NULL DEFAULT 0,
    resume_summary_json TEXT       NOT NULL DEFAULT '',
    resume_summary_text TEXT       NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS student_profile_change_requests (
    id                        BIGSERIAL PRIMARY KEY,
    student_id                BIGINT      NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    requested_by              BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    requested_student_no      VARCHAR(30)  NOT NULL DEFAULT '',
    requested_major           VARCHAR(100) NOT NULL DEFAULT '',
    requested_class_name      VARCHAR(50)  NOT NULL DEFAULT '',
    requested_graduation_year INT          NOT NULL DEFAULT 0,
    reason                    TEXT         NOT NULL DEFAULT '',
    status                    VARCHAR(20)  NOT NULL DEFAULT 'pending',
    review_comment            TEXT         NOT NULL DEFAULT '',
    reviewed_by               BIGINT REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at               TIMESTAMPTZ,
    created_at                TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- resume_chunks（长简历切块表）
CREATE TABLE IF NOT EXISTS resume_chunks (
    id          BIGSERIAL PRIMARY KEY,
    student_id  BIGINT      NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    chunk_index INT         NOT NULL,
    content     TEXT        NOT NULL,
    embedding   vector(1536) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- employments（就业信息表）
CREATE TABLE IF NOT EXISTS employments (
    id           BIGSERIAL PRIMARY KEY,
    student_id   BIGINT      NOT NULL REFERENCES students(id),
    status       VARCHAR(30) NOT NULL DEFAULT 'seeking',
                 -- seeking | interning | offered | employed | postgrad
    verification_status VARCHAR(20) NOT NULL DEFAULT 'pending',
                 -- pending | approved | rejected
    review_comment TEXT DEFAULT '',
    reviewer_id   BIGINT REFERENCES users(id) ON DELETE SET NULL,
    company_name VARCHAR(200) DEFAULT '',
    position     VARCHAR(200) DEFAULT '',
    salary_range VARCHAR(50)  DEFAULT '',
    city         VARCHAR(50)  DEFAULT '',
    offer_date   DATE,
    entry_date   DATE,
    reviewed_at  TIMESTAMPTZ,
    notes        TEXT DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS employment_evidences (
    id            BIGSERIAL PRIMARY KEY,
    employment_id BIGINT      NOT NULL REFERENCES employments(id) ON DELETE CASCADE,
    file_url      VARCHAR(500) NOT NULL,
    file_name     VARCHAR(255) NOT NULL,
    mime_type     VARCHAR(100) NOT NULL DEFAULT '',
    uploaded_by   BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- interview_sessions（面试会话表）
CREATE TABLE IF NOT EXISTS interview_sessions (
    id               BIGSERIAL PRIMARY KEY,
    chat_id          VARCHAR(255) NOT NULL UNIQUE,  -- 关联 vector_store
    student_id       BIGINT       NOT NULL REFERENCES students(id),
    title            VARCHAR(200) DEFAULT '',
    interview_type   VARCHAR(50)  DEFAULT 'general',
                     -- general | go | java | frontend | system_design
    status           VARCHAR(20)  NOT NULL DEFAULT 'ongoing', -- ongoing | completed
    total_questions  INT          DEFAULT 0,
    duration_seconds INT          DEFAULT 0,
    score            DECIMAL(5,2) DEFAULT NULL,      -- AI 总评分 0-100
    score_detail     JSONB        DEFAULT '{}'::jsonb, -- 各维度评分
    ai_summary       TEXT         DEFAULT '',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    completed_at     TIMESTAMPTZ
);

-- interview_analyses（面试分析报告表）
CREATE TABLE IF NOT EXISTS interview_analyses (
    id               BIGSERIAL PRIMARY KEY,
    session_id       BIGINT       NOT NULL REFERENCES interview_sessions(id),
    student_id       BIGINT       NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    overall_score    DECIMAL(5,2) NOT NULL DEFAULT 0,
    technical_score  DECIMAL(5,2) DEFAULT 0,   -- 技术能力
    expression_score DECIMAL(5,2) DEFAULT 0,   -- 表达能力
    logic_score      DECIMAL(5,2) DEFAULT 0,   -- 逻辑思维
    strengths        TEXT DEFAULT '[]',         -- JSON 数组
    weaknesses       TEXT DEFAULT '[]',         -- JSON 数组
    suggestions      TEXT DEFAULT '',           -- 改进建议
    detail_report    TEXT DEFAULT '',           -- Markdown 详细报告
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- personalized_suggestions（个性化建议表）
CREATE TABLE IF NOT EXISTS personalized_suggestions (
    id              BIGSERIAL PRIMARY KEY,
    student_id      BIGINT      NOT NULL REFERENCES students(id),
    session_id      BIGINT      REFERENCES interview_sessions(id) ON DELETE CASCADE,
    suggestion_type VARCHAR(50) NOT NULL, -- career | skill | interview | resume
    content         TEXT        NOT NULL,
    is_read         BOOLEAN     DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 默认管理员账号（密码：Admin@2026，首次登录请修改）
-- bcrypt hash of "Admin@2026"
INSERT INTO users (username, password_hash, role, real_name)
VALUES ('admin', '$2a$10$gQVCJdbTu/6EsEJG2usV.eK7E8oWE2qWONuPpRck1/b.KAXkKGZW2', 'admin', '系统管理员')
ON CONFLICT (username) DO NOTHING;

-- 模拟学生账号（密码统一为 Admin@2026，仅用于本地演示）
INSERT INTO users (username, password_hash, role, real_name, phone, email, status)
VALUES
    ('student_demo_01', '$2a$10$gQVCJdbTu/6EsEJG2usV.eK7E8oWE2qWONuPpRck1/b.KAXkKGZW2', 'student', '张晨曦', '13800001001', 'student_demo_01@example.com', 1),
    ('student_demo_02', '$2a$10$gQVCJdbTu/6EsEJG2usV.eK7E8oWE2qWONuPpRck1/b.KAXkKGZW2', 'student', '李沐阳', '13800001002', 'student_demo_02@example.com', 1),
    ('student_demo_03', '$2a$10$gQVCJdbTu/6EsEJG2usV.eK7E8oWE2qWONuPpRck1/b.KAXkKGZW2', 'student', '王可欣', '13800001003', 'student_demo_03@example.com', 1),
    ('student_demo_04', '$2a$10$gQVCJdbTu/6EsEJG2usV.eK7E8oWE2qWONuPpRck1/b.KAXkKGZW2', 'student', '陈思远', '13800001004', 'student_demo_04@example.com', 1),
    ('student_demo_05', '$2a$10$gQVCJdbTu/6EsEJG2usV.eK7E8oWE2qWONuPpRck1/b.KAXkKGZW2', 'student', '赵嘉宁', '13800001005', 'student_demo_05@example.com', 1),
    ('student_demo_06', '$2a$10$gQVCJdbTu/6EsEJG2usV.eK7E8oWE2qWONuPpRck1/b.KAXkKGZW2', 'student', '孙浩然', '13800001006', 'student_demo_06@example.com', 1)
ON CONFLICT (username) DO NOTHING;

INSERT INTO students (user_id, student_no, major, class_name, graduation_year, skills, self_introduction)
SELECT id, '2022201001', '软件工程', '软件工程2201', 2026, '["Go","MySQL","Docker"]', '后端开发方向，关注微服务与数据库优化。'
FROM users WHERE username = 'student_demo_01'
ON CONFLICT (student_no) DO NOTHING;

INSERT INTO students (user_id, student_no, major, class_name, graduation_year, skills, self_introduction)
SELECT id, '2022201002', '计算机科学与技术', '计科2203', 2026, '["Java","Spring","Redis"]', '偏 Java 后端，正在准备秋招面试。'
FROM users WHERE username = 'student_demo_02'
ON CONFLICT (student_no) DO NOTHING;

INSERT INTO students (user_id, student_no, major, class_name, graduation_year, skills, self_introduction)
SELECT id, '2022201003', '网络工程', '网络工程2201', 2026, '["Python","Linux","网络安全"]', '对云网络与运维自动化感兴趣。'
FROM users WHERE username = 'student_demo_03'
ON CONFLICT (student_no) DO NOTHING;

INSERT INTO students (user_id, student_no, major, class_name, graduation_year, skills, self_introduction)
SELECT id, '2022201004', '数据科学与大数据技术', '数据科学2201', 2026, '["Python","SQL","机器学习"]', '关注数据分析和机器学习工程化。'
FROM users WHERE username = 'student_demo_04'
ON CONFLICT (student_no) DO NOTHING;

INSERT INTO students (user_id, student_no, major, class_name, graduation_year, skills, self_introduction)
SELECT id, '2022201005', '人工智能', '人工智能2201', 2026, '["Python","PyTorch","大模型应用"]', '希望往 AI 应用工程方向发展。'
FROM users WHERE username = 'student_demo_05'
ON CONFLICT (student_no) DO NOTHING;

INSERT INTO students (user_id, student_no, major, class_name, graduation_year, skills, self_introduction)
SELECT id, '2022201006', '软件工程', '软件工程2202', 2027, '["Go","Kubernetes","系统设计"]', '对分布式系统和云原生方向更感兴趣。'
FROM users WHERE username = 'student_demo_06'
ON CONFLICT (student_no) DO NOTHING;

INSERT INTO student_profile_change_requests (
    student_id, requested_by, requested_student_no, requested_major,
    requested_class_name, requested_graduation_year, reason, status, created_at, updated_at
)
SELECT s.id, u.id, '2022201002', '计算机科学与技术', '计科2204', 2026,
       '申请调整班级为计科2204，已完成学院调班审批。', 'pending',
       NOW() - INTERVAL '3 day', NOW() - INTERVAL '3 day'
FROM students s
JOIN users u ON u.id = s.user_id
WHERE u.username = 'student_demo_02'
  AND NOT EXISTS (
      SELECT 1 FROM student_profile_change_requests r
      WHERE r.student_id = s.id AND r.reason = '申请调整班级为计科2204，已完成学院调班审批。'
  );

INSERT INTO student_profile_change_requests (
    student_id, requested_by, requested_student_no, requested_major,
    requested_class_name, requested_graduation_year, reason, status, created_at, updated_at
)
SELECT s.id, u.id, '2022201093', '网络工程', '网络工程2201', 2026,
       '招生办反馈学号末位登记错误，申请按教务系统修正。', 'pending',
       NOW() - INTERVAL '2 day', NOW() - INTERVAL '2 day'
FROM students s
JOIN users u ON u.id = s.user_id
WHERE u.username = 'student_demo_03'
  AND NOT EXISTS (
      SELECT 1 FROM student_profile_change_requests r
      WHERE r.student_id = s.id AND r.reason = '招生办反馈学号末位登记错误，申请按教务系统修正。'
  );

INSERT INTO student_profile_change_requests (
    student_id, requested_by, requested_student_no, requested_major,
    requested_class_name, requested_graduation_year, reason, status, created_at, updated_at
)
SELECT s.id, u.id, '2022201004', '人工智能', '数据科学2201', 2026,
       '辅修转主修审批通过，申请将专业调整为人工智能。', 'pending',
       NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'
FROM students s
JOIN users u ON u.id = s.user_id
WHERE u.username = 'student_demo_04'
  AND NOT EXISTS (
      SELECT 1 FROM student_profile_change_requests r
      WHERE r.student_id = s.id AND r.reason = '辅修转主修审批通过，申请将专业调整为人工智能。'
  );

INSERT INTO student_profile_change_requests (
    student_id, requested_by, requested_student_no, requested_major,
    requested_class_name, requested_graduation_year, reason, status, review_comment,
    reviewed_by, reviewed_at, created_at, updated_at
)
SELECT s.id, u.id, '2022201005', '人工智能', '人工智能2201', 2025,
       '误以为已满足提前毕业条件，申请将毕业年份改为 2025。', 'rejected',
       '教务系统核验后确认仍为 2026 届，当前毕业年份不予调整。',
       admin_user.id, NOW() - INTERVAL '6 hour', NOW() - INTERVAL '18 hour', NOW() - INTERVAL '6 hour'
FROM students s
JOIN users u ON u.id = s.user_id
JOIN users admin_user ON admin_user.username = 'admin'
WHERE u.username = 'student_demo_05'
  AND NOT EXISTS (
      SELECT 1 FROM student_profile_change_requests r
      WHERE r.student_id = s.id AND r.reason = '误以为已满足提前毕业条件，申请将毕业年份改为 2025。'
  );

INSERT INTO employments (
    student_id, status, verification_status, review_comment, reviewer_id,
    company_name, position, salary_range, city, offer_date, entry_date,
    reviewed_at, notes, created_at, updated_at
)
SELECT s.id, 'employed', 'approved', '已核验 offer 与入职通知，信息准确。',
       admin_user.id, '杭州云图科技有限公司', 'Go 后端开发工程师', '18k-22k x14', '杭州',
       DATE '2026-04-18', DATE '2026-07-10', NOW() - INTERVAL '5 day',
       '已确认入职，负责微服务接口开发与数据库性能优化。', NOW() - INTERVAL '12 day', NOW() - INTERVAL '5 day'
FROM students s
JOIN users u ON u.id = s.user_id
JOIN users admin_user ON admin_user.username = 'admin'
WHERE u.username = 'student_demo_01'
  AND NOT EXISTS (
      SELECT 1 FROM employments e
      WHERE e.student_id = s.id AND e.company_name = '杭州云图科技有限公司' AND e.position = 'Go 后端开发工程师'
  );

INSERT INTO employments (
    student_id, status, verification_status, review_comment,
    company_name, position, salary_range, city, offer_date, entry_date,
    notes, created_at, updated_at
)
SELECT s.id, 'interning', 'pending', '',
       '上海码川信息技术有限公司', 'Java 后端实习生', '220 元/天', '上海',
       DATE '2026-05-12', DATE '2026-05-20',
       '实习期三个月，团队有转正名额，学生已提交基础就业信息等待老师审核。', NOW() - INTERVAL '8 day', NOW() - INTERVAL '2 day'
FROM students s
JOIN users u ON u.id = s.user_id
WHERE u.username = 'student_demo_02'
  AND NOT EXISTS (
      SELECT 1 FROM employments e
      WHERE e.student_id = s.id AND e.company_name = '上海码川信息技术有限公司' AND e.position = 'Java 后端实习生'
  );

INSERT INTO employments (
    student_id, status, verification_status, review_comment, reviewer_id,
    company_name, position, salary_range, city, offer_date, entry_date,
    reviewed_at, notes, created_at, updated_at
)
SELECT s.id, 'offered', 'approved', '学院已核验 offer 截图，企业与岗位信息一致。',
       admin_user.id, '深信服科技股份有限公司', '网络安全工程师', '16k-20k x15', '深圳',
       DATE '2026-04-26', DATE '2026-07-15', NOW() - INTERVAL '4 day',
       '已拿到正式 offer，正在等待毕业后统一入职。', NOW() - INTERVAL '10 day', NOW() - INTERVAL '4 day'
FROM students s
JOIN users u ON u.id = s.user_id
JOIN users admin_user ON admin_user.username = 'admin'
WHERE u.username = 'student_demo_03'
  AND NOT EXISTS (
      SELECT 1 FROM employments e
      WHERE e.student_id = s.id AND e.company_name = '深信服科技股份有限公司' AND e.position = '网络安全工程师'
  );

INSERT INTO employments (
    student_id, status, verification_status, review_comment, reviewer_id,
    company_name, position, salary_range, city, offer_date, entry_date,
    reviewed_at, notes, created_at, updated_at
)
SELECT s.id, 'postgrad', 'approved', '已核验拟录取信息，按升学去向登记。',
       admin_user.id, '华东理工大学', '数据科学专业硕士拟录取', '', '上海',
       DATE '2026-04-09', DATE '2026-09-01', NOW() - INTERVAL '3 day',
       '学生选择继续深造，就业去向按升学统计。', NOW() - INTERVAL '14 day', NOW() - INTERVAL '3 day'
FROM students s
JOIN users u ON u.id = s.user_id
JOIN users admin_user ON admin_user.username = 'admin'
WHERE u.username = 'student_demo_04'
  AND NOT EXISTS (
      SELECT 1 FROM employments e
      WHERE e.student_id = s.id AND e.company_name = '华东理工大学' AND e.position = '数据科学专业硕士拟录取'
  );

INSERT INTO employments (
    student_id, status, verification_status, review_comment, reviewer_id,
    company_name, position, salary_range, city, offer_date, entry_date,
    reviewed_at, notes, created_at, updated_at
)
SELECT s.id, 'employed', 'rejected', '薪资证明页缺失，需补充完整 offer 或三方协议后重新提交。',
       admin_user.id, '苏州智算科技有限公司', 'AI 应用工程师', '20k-25k x14', '苏州',
       DATE '2026-05-08', DATE '2026-07-20', NOW() - INTERVAL '18 hour',
       '学生已提交就业记录，但当前佐证材料不完整，老师暂未通过审核。', NOW() - INTERVAL '6 day', NOW() - INTERVAL '18 hour'
FROM students s
JOIN users u ON u.id = s.user_id
JOIN users admin_user ON admin_user.username = 'admin'
WHERE u.username = 'student_demo_05'
  AND NOT EXISTS (
      SELECT 1 FROM employments e
      WHERE e.student_id = s.id AND e.company_name = '苏州智算科技有限公司' AND e.position = 'AI 应用工程师'
  );

INSERT INTO employments (
    student_id, status, verification_status, review_comment,
    company_name, position, salary_range, city, offer_date, entry_date,
    notes, created_at, updated_at
)
SELECT s.id, 'seeking', 'pending', '',
       '', '', '', '', NULL, NULL,
       '学生当前仍在求职，已填写求职中状态，准备参加 6 月线下面试。', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'
FROM students s
JOIN users u ON u.id = s.user_id
WHERE u.username = 'student_demo_06'
  AND NOT EXISTS (
      SELECT 1 FROM employments e
      WHERE e.student_id = s.id AND e.status = 'seeking' AND e.notes = '学生当前仍在求职，已填写求职中状态，准备参加 6 月线下面试。'
  );

-- 索引
CREATE INDEX IF NOT EXISTS idx_students_user_id ON students(user_id);
CREATE INDEX IF NOT EXISTS idx_profile_change_requests_student_id ON student_profile_change_requests(student_id);
CREATE INDEX IF NOT EXISTS idx_profile_change_requests_status ON student_profile_change_requests(status);
CREATE INDEX IF NOT EXISTS idx_employments_student_id ON employments(student_id);
CREATE INDEX IF NOT EXISTS idx_employments_verification_status ON employments(verification_status);
CREATE INDEX IF NOT EXISTS idx_interview_sessions_student_id ON interview_sessions(student_id);
CREATE INDEX IF NOT EXISTS idx_interview_analyses_student_id ON interview_analyses(student_id);
CREATE INDEX IF NOT EXISTS idx_interview_sessions_chat_id ON interview_sessions(chat_id);
CREATE INDEX IF NOT EXISTS idx_suggestions_student_id ON personalized_suggestions(student_id);
CREATE INDEX IF NOT EXISTS idx_suggestions_session_id ON personalized_suggestions(session_id);
DELETE FROM personalized_suggestions WHERE session_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_employment_evidences_employment_id ON employment_evidences(employment_id);
CREATE INDEX IF NOT EXISTS idx_resume_chunks_student_id ON resume_chunks(student_id);
CREATE INDEX IF NOT EXISTS idx_resume_chunks_embedding_hnsw
    ON resume_chunks USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
