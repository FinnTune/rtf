CREATE TABLE user (
 id INTEGER NOT NULL PRIMARY KEY,
 fname TEXT NOT NULL,
 lname TEXT NOT NULL,
 uname TEXT NOT NULL,
 email TEXT NOT NULL,
 age TEXT NOT NULL,
 gender TEXT NOT NULL,
 pass TEXT NOT NULL,
 created_at TEXT NOT NULL
);

CREATE TABLE category (
 id INTEGER NOT NULL PRIMARY KEY,
 category_name VARCHAR(30) NOT NULL,
 descript VARCHAR(100),
 created_at DATETIME NOT NULL
);

CREATE TABLE post (
 id INTEGER NOT NULL PRIMARY KEY,
 user_id INTEGER NOT NULL,
 title VARCHAR(30) NOT NULL,
 content TEXT NOT NULL,
 created_at DATETIME NOT NULL,
 updated_at DATETIME NOT NULL,
 liked_no INTEGER,
 disliked_no INTEGER,
 img_url VARCHAR(100),
 approved TINYINT(1) NOT NULL,
 dummy TINYINT(1) NOT NULL,
 FOREIGN KEY(user_id) REFERENCES user(id)
);

CREATE TABLE comment (
 id INTEGER NOT NULL PRIMARY KEY,
 user_id INTEGER NOT NULL,
 post_id INTEGER NOT NULL,
 content TEXT NOT NULL,
 created_at DATETIME NOT NULL,
 updated_at DATETIME NOT NULL,
 liked_no INTEGER,
 disliked_no INTEGER,
 FOREIGN KEY(user_id) REFERENCES user(id),
 FOREIGN KEY(post_id) REFERENCES post(id)
);

CREATE TABLE user_post_reaction (
 id INTEGER NOT NULL PRIMARY KEY,
 user_id INTEGER NOT NULL,
 post_id INTEGER NOT NULL,
 is_liked TINYINT(1) NOT NULL,
 created_at DATETIME NOT NULL,
 FOREIGN KEY(user_id) REFERENCES user(id),
 FOREIGN KEY(post_id) REFERENCES post(id)
);

CREATE TABLE user_comment_reaction (
 id INTEGER NOT NULL PRIMARY KEY,
 user_id INTEGER NOT NULL,
 comment_id INTEGER NOT NULL,
 is_liked TINYINT(1) NOT NULL,
 created_at DATETIME NOT NULL,
 FOREIGN KEY(user_id) REFERENCES user(id),
 FOREIGN KEY(comment_id) REFERENCES comment(id)
);

CREATE TABLE category_relation (
 id INTEGER NOT NULL PRIMARY KEY,
 category_id INTEGER NOT NULL,
 post_id INTEGER NOT NULL,
 FOREIGN KEY(category_id) REFERENCES category(id),
 FOREIGN KEY(post_id) REFERENCES post(id)
);

CREATE TABLE message (
 id INTEGER NOT NULL PRIMARY KEY,
 from_user INTEGER NOT NULL,
 to_user INTEGER NOT NULL,
 is_read TINYINT(1) NOT NULL,
 message TEXT NOT NULL,
 created_at DATETIME NOT NULL,
 FOREIGN KEY(from_user) REFERENCES user(id),
 FOREIGN KEY(to_user) REFERENCES user(id)
);


-- INSERT INTO user (id, username, passwrd, email, fname, lname, age, gender, created_at)
-- VALUES
--     (1, 'admin', 'admin', 'admin@admin.com', 'fname', 'lname', 99, 'male', DateTime('now', 'localtime')),
--     (2, 'user', 'user', 'user@user.com', 'fname', 'lname', 11, 'female', DateTime('now', 'localtime')),
--     (3, 'user2', 'user2', 'user2@user2.com', 'fname2', 'lname2', 12, 'female', DateTime('now', 'localtime')),
--     (4, 'user3', 'user3', 'user3@user3.com', 'fname3', 'lname3', 13, 'female', DateTime('now', 'localtime'));

-- INSERT INTO category (id,category_name,descript,created_at)
-- VALUES
--     (1,'Cuisines','Recommendation regarding food in Mariehamn',DateTime('now','localtime')),
--     (2,'Places','Places worth a visit in Mariehamn',DateTime('now','localtime')),
--     (3,'Activities','Interesting events happening in Mariehamn',DateTime('now','localtime'));

-- INSERT INTO post (id,user_id,title,content,created_at,updated_at,liked_no,disliked_no,img_url,approved,dummy)
-- VALUES
--     (1,1,'Welcome to the Cuisines category!','Be the first to post in this category!',DateTime('now','localtime'),DateTime('now','localtime'),0,0,'',0,1),
--     (2,2,'Welcome to the Places category!','Be the first to post in this category!',DateTime('now','localtime'),DateTime('now','localtime'),0,0,'',0,1),
--     (3,3,'Welcome to the Activities category!','Be the first to post in this category!',DateTime('now','localtime'),DateTime('now','localtime'),0,0,'',0,1),
--     (4,2,'Asian Food','Thai Khun Mom serves very typical Asian food in Mariehamn',DateTime('now','localtime'),DateTime('now','localtime'),0,0,'',1,0),
--     (5,3,'Swedish Class','Swedish class occurs every Tuesday and Thursday from 4pm',DateTime('now','localtime'),DateTime('now','localtime'),0,0,'',1,0),
--     (6,4,'Best Sushi','Fina Fisken is the best sushi in Mariehamn',DateTime('now','localtime'),DateTime('now','localtime'),0,0,'',1,0),
--     (7,5,'Poker Night','Poker Game Night occurs every Friday from 8pm',DateTime('now','localtime'),DateTime('now','localtime'),0,0,'',1,0),
--     (8,1,'Real Embassy','Brazilian Real Embassy is now in Mariehamn',DateTime('now','localtime'),DateTime('now','localtime'),0,0,'',1,0);

-- INSERT INTO category_relation (id,category_id,post_id)
-- VALUES
--     (1,1,1),
--     (2,2,2),
--     (3,3,3),
--     (4,1,4),
--     (5,1,6),
--     (6,2,4),
--     (7,2,6),
--     (8,2,8),
--     (9,3,5),
--     (10,3,7);

-- INSERT INTO message (id,from_user,to_user,is_read,message,created_at)
-- VALUES
--     (1,1,2,1,'Hello user!',DateTime('now','localtime')),
--     (2,2,1,1,'Hello admin!',DateTime('now','localtime')),
--     (3,1,3,1,'Hello user2!',DateTime('now','localtime')),
--     (4,3,1,1,'Hello admin!',DateTime('now','localtime')),
--     (5,1,4,1,'Hello user3!',DateTime('now','localtime')),
--     (6,4,1,1,'Hello admin!',DateTime('now','localtime'));