CREATE TABLE orders
(
    id UInt64,
    method Enum('INSERT' = 1, 'UPDATE' = 2, 'DELETE' = 3),
    created_at DateTime64(6, 'UTC'),
    -- Поля из вложенного JSON в data
    status LowCardinality(String),
    user_id UInt64,
    order_id UInt64,
    quantity UInt32,
    order_created_at DateTime64(6, 'UTC'), -- переименовываем created_at из вложенного JSON
    order_date DateTime64(6, 'UTC'),
    product_id UInt64,
    order_price Decimal64(2),
    total_amount Decimal64(2),
    payment_method LowCardinality(String)
) ENGINE = MergeTree ORDER BY (method, order_id, created_at);

-- Альтернативный вариант с использованием String для хранения JSON данных
CREATE TABLE orders_json
(
    id UInt64,
    method Enum('INSERT' = 1, 'UPDATE' = 2, 'DELETE' = 3),
    created_at DateTime64(6, 'UTC'),
    data String
) ENGINE = MergeTree ORDER BY (id);

-- Удаляем старые объекты если существуют
DROP TABLE IF EXISTS orders_queue;
DROP TABLE IF EXISTS orders_mv;

-- Таблица для потребления данных из Kafka
CREATE TABLE orders_queue
(
    id UInt64,
    method Enum('INSERT' = 1, 'UPDATE' = 2, 'DELETE' = 3),
    created_at DateTime64(6, 'UTC'),
    data String
)
ENGINE = Kafka('127.0.0.1:9092', 'orders', 'clickhouse_consumer_group_local',
        'JSONEachRow')
SETTINGS kafka_thread_per_consumer = 0,
         kafka_num_consumers = 1,
         kafka_skip_broken_messages = 10,
         kafka_max_block_size = 1000,
         stream_like_engine_allow_direct_select = 1;

-- Материализованное представление для переноса данных из Kafka в основную таблицу
CREATE MATERIALIZED VIEW orders_mv TO orders AS
SELECT
    id,
    method,
    created_at,
    JSONExtractString(data, 'status') AS status,
    JSONExtractUInt(data, 'user_id') AS user_id,
    JSONExtractUInt(data, 'order_id') AS order_id,
    JSONExtractUInt(data, 'quantity') AS quantity,
    toDateTime64(JSONExtractString(data, 'created_at'), 6, 'UTC') AS order_created_at,
    toDateTime64(JSONExtractString(data, 'order_date'), 6, 'UTC') AS order_date,
    JSONExtractUInt(data, 'product_id') AS product_id,
    JSONExtractFloat(data, 'order_price') AS order_price,
    JSONExtractFloat(data, 'total_amount') AS total_amount,
    JSONExtractString(data, 'payment_method') AS payment_method
FROM orders_queue;

-- Проверка таблицы-очереди (выполнить после создания)
-- SELECT * FROM orders_queue LIMIT 10 SETTINGS stream_like_engine_allow_direct_select = 1;

-- Остановка и перезапуск потребления (при необходимости)
-- DETACH TABLE orders_queue;
-- ATTACH TABLE orders_queue;

-- Сброс смещений в Kafka (крайний случай)
-- ALTER TABLE orders_queue RESET STREAM;