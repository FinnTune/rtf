CREATE TABLE user (
 id INTEGER NOT NULL PRIMARY KEY,
 fname VARCHAR(30) NOT NULL,
 lname VARCHAR(30) NOT NULL,
 uname VARCHAR(30) NOT NULL,
 email VARCHAR(30) NOT NULL,
 age VARCHAR(3) NOT NULL,
 gender VARCHAR(10) NOT NULL,
 pass VARCHAR(20) NOT NULL,
 created_at VARCHAR(30) NOT NULL
);

CREATE TABLE category (
 id INTEGER NOT NULL PRIMARY KEY,
 category_name VARCHAR(30) NOT NULL
);

CREATE TABLE post (
 id INTEGER NOT NULL PRIMARY KEY,
 user_id INTEGER NOT NULL,
 title VARCHAR(30) NOT NULL,
 content VARCHAR(150) NOT NULL,
 category VARCHAR(15) NOT NULL,
 category_id INTEGER NOT NULL,
 created_at DATETIME NOT NULL,
--  liked_no INTEGER,
--  disliked_no INTEGER,
--  img_url VARCHAR(100),
--  approved TINYINT(1) NOT NULL,
--  dummy TINYINT(1) NOT NULL,
FOREIGN KEY(category_id) REFERENCES category(id),
FOREIGN KEY(user_id) REFERENCES user(id)
);

CREATE TABLE comment (
 id INTEGER NOT NULL PRIMARY KEY,
 user_id INTEGER NOT NULL,
 post_id INTEGER NOT NULL,
 content VARCHAR(150) NOT NULL,
 created_at DATETIME NOT NULL,
--  updated_at DATETIME NOT NULL,
--  liked_no INTEGER,
--  disliked_no INTEGER,
 FOREIGN KEY(user_id) REFERENCES user(id),
 FOREIGN KEY(post_id) REFERENCES post(id)
);

-- CREATE TABLE user_post_reaction (
--  id INTEGER NOT NULL PRIMARY KEY,
--  user_id INTEGER NOT NULL,
--  post_id INTEGER NOT NULL,
--  is_liked TINYINT(1) NOT NULL,
--  created_at DATETIME NOT NULL,
--  FOREIGN KEY(user_id) REFERENCES user(id),
--  FOREIGN KEY(post_id) REFERENCES post(id)
-- );

-- CREATE TABLE user_comment_reaction (
--  id INTEGER NOT NULL PRIMARY KEY,
--  user_id INTEGER NOT NULL,
--  comment_id INTEGER NOT NULL,
--  is_liked TINYINT(1) NOT NULL,
--  created_at DATETIME NOT NULL,
--  FOREIGN KEY(user_id) REFERENCES user(id),
--  FOREIGN KEY(comment_id) REFERENCES comment(id)
-- );

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
 txt TEXT NOT NULL,
 created_at DATETIME NOT NULL,
 FOREIGN KEY(from_user) REFERENCES user(id),
 FOREIGN KEY(to_user) REFERENCES user(id)
);

-- Insert in user table for testing
-- INSERT INTO user ( id, fname, lname, uname, email, age, gender, pass, created_at) VALUES (1, admin, admin, admin, admin@example.com, 1, male, passHash!!!,  DateTime('now', 'localtime'))


-- INSERT INTO user (id, username, passwrd, email, fname, lname, age, gender, created_at)
-- VALUES
--     (1, 'admin', 'admin', 'admin@admin.com', 'fname', 'lname', 99, 'male', DateTime('now', 'localtime')),
--     (2, 'user', 'user', 'user@user.com', 'fname', 'lname', 11, 'female', DateTime('now', 'localtime')),
--     (3, 'user2', 'user2', 'user2@user2.com', 'fname2', 'lname2', 12, 'female', DateTime('now', 'localtime')),
--     (4, 'user3', 'user3', 'user3@user3.com', 'fname3', 'lname3', 13, 'female', DateTime('now', 'localtime'));

INSERT INTO category (id,category_name)
VALUES
    (1,'Cuisine'),
    (2,'Places'),
    (3,'Activities'),
    (4,'Events'),
    (5,'Code'),
    (6,'Language'),
    (7,'Sports'),
    (8,'Politics'),
    (9,'Social'),
    (10,'Religion'),
    (11,'Business'),
    (12,'Geography'),
    (13,'Science'),
    (14,'Health'),
    (15,'Other');

INSERT INTO post (user_id,title,content,category,category_id,created_at)
VALUES
    (1,'Welcome to the Cuisines category!','Be the first to post in this category!','Other',15,DateTime('now','localtime')),
    (1,'Welcome to the Places category!','Be the first to post in this category!','Other',15,DateTime('now','localtime')),
    (1,'Welcome to the Activities category!','Be the first to post in this category!','Other',15,DateTime('now','localtime')),
    (1,'Asian Food','Thai Khun Mom serves very typical Asian food in Mariehamn','Cuisine',1,DateTime('now','localtime')),
    (1,'Swedish Class','Swedish class occurs every Tuesday and Thursday from 4pm','Language',6,DateTime('now','localtime')),
    (1,'Best Sushi','Fina Fisken is the best sushi in Mariehamn','Cuisine',1,DateTime('now','localtime')),
    (1,'Poker Night','Poker Game Night occurs every Friday from 8pm','Social',9,DateTime('now','localtime')),
    (1,'Real Embassy','Brazilian Real Embassy is now in Mariehamn','Politics',8,DateTime('now','localtime'));

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