import { showMessage, setButtonLoading } from "./notify.js";

export function getAllPosts() {
    console.log("Getting all posts.")
    fetch('getAllPosts', {
        method: 'GET',
        mode: 'cors',
        headers: {
            'Content-Type': 'application/json'
        }}
    ).then((response) => {
        if(response.ok){
        console.log("Received all posts.")
        // Arrange posts in descending order by date created
        if (document.getElementById('posts')) {
            clearTable();
        } else {
            createPostsTable();
        }
        let posts = response.json();
        console.log("PostsBef:", posts);
        return posts;
        }
        return response.text().then((message) => {
            throw new Error(message || `Failed to load posts (${response.status})`);
        });
    }).then((posts) => {
        console.log("PostsAft: ", posts)
        posts.sort((a, b) => (a.CreatedAt > b.CreatedAt) ? -1 : 1);
        let table = document.getElementById('posts-table');
        let tbody = table.querySelector('tbody');
        if (posts.length === 0) {
            showEmptyState(tbody, "No posts yet — be the first to post!");
            return;
        }
        for(let i = 0; i < posts.length; i++){
            let row = tbody.insertRow();
            let title = row.insertCell(0);
            let content = row.insertCell(1);
            let author = row.insertCell(2);
            let dateCreated = row.insertCell(3);
            let link = document.createElement("a");
            link.href = "/posts/" + posts[i].Id;
            link.className = "post-link";
            link.textContent = posts[i].Title;
            link.addEventListener("click", function(event){
                event.preventDefault();
                displaySinglePost(posts[i]);
            });
            title.appendChild(link);
            content.textContent = posts[i].Content;
            author.textContent = posts[i].Author;
            dateCreated.textContent = posts[i].Created;
    }}).catch((error) => {
        showMessage("Err: " + error.message, "error");
        console.log("Err: ", error);
    });

    return false;
}

export function showEmptyState(tbody, message) {
    const row = tbody.insertRow();
    const cell = row.insertCell(0);
    cell.colSpan = 4;
    cell.className = 'empty-state';
    cell.textContent = message;
}

export function createPostsTable() {
    // Get the main content element
    const mainContent = document.getElementById('main-content');
  
    // Create the posts div element
    const postsDiv = document.createElement('div');
    postsDiv.setAttribute('id', 'posts');
  
    // Create the heading element
    const heading = document.createElement('h3');
    heading.textContent = 'Latest Posts';
  
    // Create the table element
    const table = document.createElement('table');
    table.setAttribute('id', 'posts-table');
  
    // Create the table header row and cells
    const thead = document.createElement('thead'); // Create thead element
    const headerRow = document.createElement('tr');
    const titleHeader = document.createElement('th');
    titleHeader.textContent = 'Title';
    const contentHeader = document.createElement('th');
    contentHeader.textContent = 'Content';
    const authorHeader = document.createElement('th');
    authorHeader.textContent = 'Author';
    const createdHeader = document.createElement('th');
    createdHeader.textContent = 'Created';
  
    // Append the cells to the header row
    headerRow.appendChild(titleHeader);
    headerRow.appendChild(contentHeader);
    headerRow.appendChild(authorHeader);
    headerRow.appendChild(createdHeader);
  
    // Append the header row to the thead element
    thead.appendChild(headerRow);
  
    // Create the table body element
    const tbody = document.createElement('tbody');
  
    // Append the thead and tbody to the table
    table.appendChild(thead);
    table.appendChild(tbody);
  
    // Append the heading and table to the posts div
    postsDiv.appendChild(heading);
    postsDiv.appendChild(table);
  
    // Append the posts div to the main content element
    mainContent.appendChild(postsDiv);
  }
  
  
export function clearTable() {
    const tableBody = document.querySelector('#posts-table tbody');
    tableBody.innerHTML = '';
}

export async function displaySinglePost(post) {
    console.log("Displaying single post.", post);
    let mainContent = document.getElementById("main-content");
    let singlePostDiv = document.createElement("div");
    singlePostDiv.id = "single-post";
    let title = document.createElement("h3");
    title.textContent = post.Title;
    let content = document.createElement("p");
    content.textContent = post.Content;
    let author = document.createElement("p");
    author.textContent = "Author: " + post.Author;
    let dateCreated = document.createElement("p");
    dateCreated.textContent = "Created: " + post.Created;
    let backButton = document.createElement("button");
    backButton.className = "btns";
    backButton.textContent = "Back to Posts";
    backButton.addEventListener("click", function(event){
        event.preventDefault();
        mainContent.innerHTML = "";
        getAllPosts();
    });
    singlePostDiv.appendChild(title);
    singlePostDiv.appendChild(content);
    singlePostDiv.appendChild(author);
    singlePostDiv.appendChild(dateCreated);

    // Only the post's author gets Edit/Delete controls.
    let postActions = document.createElement("div");
    postActions.id = "post-actions";
    if (post.Author === localStorage.getItem('username')) {
        let editButton = document.createElement("button");
        editButton.type = "button";
        editButton.className = "btns";
        editButton.textContent = "Edit";

        let deleteButton = document.createElement("button");
        deleteButton.type = "button";
        deleteButton.className = "btns";
        deleteButton.textContent = "Delete";

        editButton.addEventListener("click", () => {
            let titleInput = document.createElement("input");
            titleInput.type = "text";
            titleInput.value = post.Title;
            titleInput.maxLength = 100;
            titleInput.required = true;

            let contentInput = document.createElement("textarea");
            contentInput.value = post.Content;
            contentInput.maxLength = 2000;
            contentInput.required = true;
            contentInput.rows = 4;
            contentInput.cols = 50;

            let saveButton = document.createElement("button");
            saveButton.type = "button";
            saveButton.className = "btns";
            saveButton.textContent = "Save";

            let cancelButton = document.createElement("button");
            cancelButton.type = "button";
            cancelButton.className = "btns";
            cancelButton.textContent = "Cancel";

            let editForm = document.createElement("div");
            editForm.id = "edit-post-form";
            editForm.appendChild(titleInput);
            editForm.appendChild(contentInput);
            editForm.appendChild(saveButton);
            editForm.appendChild(cancelButton);

            title.style.display = "none";
            content.style.display = "none";
            postActions.style.display = "none";
            singlePostDiv.insertBefore(editForm, author);

            cancelButton.addEventListener("click", () => {
                editForm.remove();
                title.style.display = "";
                content.style.display = "";
                postActions.style.display = "";
            });

            saveButton.addEventListener("click", async () => {
                const newTitle = titleInput.value.trim();
                const newContent = contentInput.value.trim();
                if (!newTitle || !newContent) {
                    return;
                }
                setButtonLoading(saveButton, true, "Saving...");
                try {
                    const updated = await editPostRequest(post.PostId, newTitle, newContent);
                    showMessage("Post updated.", "success");
                    displaySinglePost({ ...post, Title: updated.title, Content: updated.content });
                } catch (error) {
                    showMessage("Err: " + error.message, "error");
                    console.log("Err: ", error);
                    setButtonLoading(saveButton, false);
                }
            });
        });

        deleteButton.addEventListener("click", async () => {
            if (!confirm("Delete this post? This cannot be undone.")) {
                return;
            }
            setButtonLoading(deleteButton, true, "Deleting...");
            try {
                await deletePostRequest(post.PostId);
                showMessage("Post deleted.", "success");
                mainContent.innerHTML = "";
                getAllPosts();
            } catch (error) {
                showMessage("Err: " + error.message, "error");
                console.log("Err: ", error);
                setButtonLoading(deleteButton, false);
            }
        });

        postActions.appendChild(editButton);
        postActions.appendChild(deleteButton);
    }
    singlePostDiv.appendChild(postActions);
    singlePostDiv.appendChild(backButton);
    mainContent.innerHTML = "";
    mainContent.appendChild(singlePostDiv);

    // Create a comments section
    let commentsSection = document.createElement("div");
    commentsSection.id = "comments-section";
    let commentsHeading = document.createElement("h4");
    commentsHeading.textContent = "Comments:";
    commentsSection.appendChild(commentsHeading);

    // Fetch comments for the post
    try {
        let comments = await fetchComments(post.PostId);
        console.log("Comments fetch: ", comments)
        comments.forEach(comment => {
            renderComment(comment, commentsSection);
        });
    } catch (error) {
        showMessage("Err: " + error.message, "error");
        console.log("Err: ", error);
    }

    // Create a form to submit a new comment
    let commentForm = document.createElement("form");
    let commentInput = document.createElement("input");
    commentInput.type = "text";
    commentInput.name = "comment";
    commentInput.placeholder = "Enter your comment here";
    commentInput.maxLength = 500;
    commentInput.required = true;
    let submitButton = document.createElement("button");
    submitButton.type = "submit";
    submitButton.textContent = "Submit Comment";
    commentForm.appendChild(commentInput);
    commentForm.appendChild(submitButton);

    // Add an event listener to handle form submission
    commentForm.addEventListener("submit", async function (event) {
        console.log("Submitting comment.", commentInput.value);
        event.preventDefault();
        let commentContent = commentInput.value.trim();
        if (commentContent) {
            setButtonLoading(submitButton, true, 'Posting...');
            try {
                await submitComment(post.PostId, commentContent);
                commentInput.value = "";
                commentsSection.replaceChildren();
                let refreshedHeading = document.createElement("h4");
                refreshedHeading.textContent = "Comments:";
                commentsSection.appendChild(refreshedHeading);
                let updatedComments = await fetchComments(post.PostId);
                updatedComments.forEach(comment => {
                    renderComment(comment, commentsSection);
                });
            } catch (error) {
                showMessage("Err: " + error.message, "error");
                console.log("Err: ", error);
            } finally {
                setButtonLoading(submitButton, false);
            }
        }
    });

    // Add comments section and form to the singlePostDiv
    singlePostDiv.appendChild(commentsSection);
    singlePostDiv.appendChild(commentForm);
}

async function fetchComments(postId) {
    const response = await fetch(`/comments?postId=${postId}`);
    if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to load comments (${response.status})`);
    }
    const comments = await response.json();
    console.log("Comments: ", comments)
    return comments;
}

async function submitComment(postId, commentContent) {
    const response = await fetch('/addcomment', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ post_id: postId, content: commentContent })
    });
    if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to submit comment (${response.status})`);
    }
    const result = await response.json();
    return result;
}

async function editPostRequest(id, title, content) {
    const response = await fetch('/editPost', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id, title, content })
    });
    if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to update post (${response.status})`);
    }
    return response.json();
}

async function deletePostRequest(id) {
    const response = await fetch('/deletePost', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id })
    });
    if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to delete post (${response.status})`);
    }
}

async function editCommentRequest(id, content) {
    const response = await fetch('/editComment', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id, content })
    });
    if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to update comment (${response.status})`);
    }
    return response.json();
}

async function deleteCommentRequest(id) {
    const response = await fetch('/deleteComment', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id })
    });
    if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to delete comment (${response.status})`);
    }
}

// Renders a single comment, with inline Edit/Delete controls when the
// current user is the comment's author.
function renderComment(comment, container) {
    let commentElement = document.createElement("p");
    commentElement.className = "comment";
    let textSpan = document.createElement("span");
    textSpan.textContent = comment.username + ": " + comment.content;
    commentElement.appendChild(textSpan);

    if (comment.username === localStorage.getItem('username')) {
        let editButton = document.createElement("button");
        editButton.type = "button";
        editButton.className = "btns";
        editButton.textContent = "Edit";

        let deleteButton = document.createElement("button");
        deleteButton.type = "button";
        deleteButton.className = "btns";
        deleteButton.textContent = "Delete";

        editButton.addEventListener("click", () => {
            let input = document.createElement("input");
            input.type = "text";
            input.value = comment.content;
            input.maxLength = 500;
            input.required = true;

            let saveButton = document.createElement("button");
            saveButton.type = "button";
            saveButton.className = "btns";
            saveButton.textContent = "Save";

            let cancelButton = document.createElement("button");
            cancelButton.type = "button";
            cancelButton.className = "btns";
            cancelButton.textContent = "Cancel";

            commentElement.replaceChildren(input, saveButton, cancelButton);
            input.focus();

            cancelButton.addEventListener("click", () => {
                commentElement.replaceChildren(textSpan, editButton, deleteButton);
            });

            saveButton.addEventListener("click", async () => {
                const newContent = input.value.trim();
                if (!newContent) {
                    return;
                }
                setButtonLoading(saveButton, true, "Saving...");
                try {
                    await editCommentRequest(comment.id, newContent);
                    comment.content = newContent;
                    textSpan.textContent = comment.username + ": " + comment.content;
                    commentElement.replaceChildren(textSpan, editButton, deleteButton);
                    showMessage("Comment updated.", "success");
                } catch (error) {
                    showMessage("Err: " + error.message, "error");
                    console.log("Err: ", error);
                    setButtonLoading(saveButton, false);
                }
            });
        });

        deleteButton.addEventListener("click", async () => {
            if (!confirm("Delete this comment? This cannot be undone.")) {
                return;
            }
            setButtonLoading(deleteButton, true, "Deleting...");
            try {
                await deleteCommentRequest(comment.id);
                commentElement.remove();
                showMessage("Comment deleted.", "success");
            } catch (error) {
                showMessage("Err: " + error.message, "error");
                console.log("Err: ", error);
                setButtonLoading(deleteButton, false);
            }
        });

        commentElement.appendChild(editButton);
        commentElement.appendChild(deleteButton);
    }

    container.appendChild(commentElement);
}