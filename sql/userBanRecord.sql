create table user_ban_record
(
    account    varchar(255) not null
        primary key,
    server_id int          not null,
    reason     int          not null,
    start_time bigint       not null,
    end_time   bigint       null
);
