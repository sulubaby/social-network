create table if not exists group_invitations (
   group_id   integer not null,
   user_id    integer not null,
   inviter_id integer not null,
   status     text not null default 'pending',
   created_at datetime not null default current_timestamp,
   primary key ( group_id,
                 user_id ),
   foreign key ( group_id )
      references groups ( id )
         on delete cascade,
   foreign key ( user_id )
      references user ( id )
         on delete cascade,
   foreign key ( inviter_id )
      references user ( id )
         on delete cascade
);