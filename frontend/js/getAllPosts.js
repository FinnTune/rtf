import { showMessage, setButtonLoading } from "./notify.js";
import { getCategories } from "./categories.js";
import { setActiveNav } from "./navigation.js";
import { showAuthorPosts } from "./authorPosts.js";

export const POSTS_PAGE_SIZE = 10;
const COMMENTS_PAGE_SIZE = 20;

export function getAllPosts(offset = 0) {
    fetch(`getAllPosts?limit=${POSTS_PAGE_SIZE}&offset=${offset}`, {
        method: 'GET',
        mode: 'cors',
        headers: {
            'Content-Type': 'application/json'
        }}
    ).then((response) => {
        if(response.ok){
        if (document.getElementById('posts')) {
            clearTable();
        } else {
            createPostsTable();
        }
        const total = parseInt(response.headers.get('X-Total-Count') || '0', 10);
        return response.json().then((posts) => ({ posts, total }));
        }
        return response.text().then((message) => {
            throw new Error(message || `Failed to load posts (${response.status})`);
        });
    }).then(({ posts, total }) => {
        // Posts already arrive newest-first from the server; no client-side
        // sort needed (and none is possible — the API doesn't return a
        // CreatedAt field, only Created).
        if (offset > 0 && posts.length === 0) {
            // The page we asked for is now empty (e.g. the last post on it
            // was deleted) — snap back to the previous page instead of
            // showing a dead end.
            getAllPosts(Math.max(0, offset - POSTS_PAGE_SIZE));
            return;
        }
        let postList = document.getElementById('posts-table');
        if (posts.length === 0) {
            showEmptyState(postList, "No posts yet — be the first to post!");
        } else {
            renderPostRows(postList, posts);
        }
        renderPagination(offset, POSTS_PAGE_SIZE, total, getAllPosts);
    }).catch((error) => {
        showMessage("Err: " + error.message, "error");
        console.error(error);
    });

    return false;
}

// Renders Previous/Next controls into #posts-pagination. `onNavigate(newOffset)`
// is called to load the requested page — shared by the all-posts and
// category-filter views, each of which knows how to (re)fetch its own data.
export function renderPagination(offset, limit, total, onNavigate) {
    const paginationDiv = document.getElementById('posts-pagination');
    if (!paginationDiv) {
        return;
    }
    paginationDiv.replaceChildren();

    if (total === 0) {
        return;
    }

    const prevButton = document.createElement('button');
    prevButton.type = 'button';
    prevButton.className = 'btns';
    prevButton.textContent = 'Previous';
    prevButton.disabled = offset === 0;
    prevButton.addEventListener('click', () => onNavigate(Math.max(0, offset - limit)));

    const hasMore = offset + limit < total;
    const nextButton = document.createElement('button');
    nextButton.type = 'button';
    nextButton.className = 'btns';
    nextButton.textContent = 'Next';
    nextButton.disabled = !hasMore;
    nextButton.addEventListener('click', () => onNavigate(offset + limit));

    const rangeLabel = document.createElement('span');
    rangeLabel.className = 'pagination-range';
    const rangeStart = offset + 1;
    const rangeEnd = Math.min(offset + limit, total);
    rangeLabel.textContent = ` ${rangeStart}-${rangeEnd} of ${total} `;

    paginationDiv.appendChild(prevButton);
    paginationDiv.appendChild(rangeLabel);
    paginationDiv.appendChild(nextButton);
}

export function showEmptyState(container, message) {
    const item = document.createElement('li');
    item.className = 'empty-state';
    item.textContent = message;
    container.appendChild(item);
}

// Renders post cards into the existing #posts-table list. Shared by the all
// posts, category filter, and search views so the card markup and the
// single-post click handler only live in one place.
export function renderPostRows(container, posts) {
    for (let i = 0; i < posts.length; i++) {
        let card = document.createElement("li");
        card.className = "post-card";

        let title = document.createElement("div");
        title.className = "post-card-title";
        let link = document.createElement("a");
        link.href = "/posts/" + posts[i].PostId;
        link.textContent = posts[i].Title;
        link.addEventListener("click", function(event){
            event.preventDefault();
            displaySinglePost(posts[i]);
        });
        title.appendChild(link);

        let preview = document.createElement("div");
        preview.className = "post-card-preview";
        preview.textContent = posts[i].Content;

        let meta = document.createElement("div");
        meta.className = "post-card-meta";
        let authorLink = document.createElement("a");
        authorLink.href = "#";
        authorLink.className = "author-link";
        authorLink.textContent = posts[i].Author;
        authorLink.addEventListener("click", function(event){
            event.preventDefault();
            showAuthorPosts(posts[i].Author);
        });
        meta.appendChild(authorLink);
        meta.appendChild(document.createTextNode(" · " + posts[i].Created));

        card.appendChild(title);
        card.appendChild(preview);
        card.appendChild(meta);
        container.appendChild(card);
    }
}

export function createPostsTable(headingText = 'Latest Posts') {
    // Get the main content element
    const mainContent = document.getElementById('main-content');

    // Create the posts div element
    const postsDiv = document.createElement('div');
    postsDiv.setAttribute('id', 'posts');

    // Create the heading element
    const heading = document.createElement('h3');
    heading.textContent = headingText;

    // Create the post list element
    const list = document.createElement('ul');
    list.className = 'post-list';
    list.setAttribute('id', 'posts-table');

    // Append the heading and list to the posts div
    postsDiv.appendChild(heading);
    postsDiv.appendChild(list);

    // Append the posts div to the main content element
    mainContent.appendChild(postsDiv);

    // Pagination controls, populated/updated by renderPagination(). Kept as
    // a sibling of #posts so Prev/Next stay visible regardless of how many
    // cards the list holds.
    const paginationDiv = document.createElement('div');
    paginationDiv.id = 'posts-pagination';
    mainContent.appendChild(paginationDiv);
  }


export function clearTable() {
    const list = document.getElementById('posts-table');
    list.innerHTML = '';
}

export async function displaySinglePost(post) {
    // Reflect the open post in the URL so it can be bookmarked/shared and
    // survives a page refresh (see getPost() + main.js's deep-link bootstrap).
    const targetPath = "/posts/" + post.PostId;
    if (window.location.pathname !== targetPath) {
        history.pushState({}, '', targetPath);
    }
    setActiveNav(null);
    let mainContent = document.getElementById("main-content");
    let singlePostDiv = document.createElement("div");
    singlePostDiv.id = "single-post";
    let title = document.createElement("h3");
    title.textContent = post.Title;
    let content = document.createElement("p");
    content.textContent = post.Content;
    let author = document.createElement("p");
    author.appendChild(document.createTextNode("Author: "));
    let authorLink = document.createElement("a");
    authorLink.href = "#";
    authorLink.className = "author-link";
    authorLink.textContent = post.Author;
    authorLink.addEventListener("click", function(event){
        event.preventDefault();
        showAuthorPosts(post.Author);
    });
    author.appendChild(authorLink);
    let dateCreated = document.createElement("p");
    dateCreated.textContent = "Created: " + post.Created;
    let backButton = document.createElement("button");
    backButton.className = "btns";
    backButton.textContent = "Back to Posts";
    backButton.addEventListener("click", function(event){
        event.preventDefault();
        history.pushState({}, '', '/');
        mainContent.innerHTML = "";
        setActiveNav('all-posts-button');
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

        editButton.addEventListener("click", async () => {
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

            let categoriesDiv = document.createElement("div");
            categoriesDiv.id = "edit-post-categories";
            let checkboxes = [];
            try {
                const [allCategories, postCategories] = await Promise.all([
                    getCategories(),
                    getPostCategories(post.PostId),
                ]);
                const currentIds = new Set(postCategories.map((c) => c.id));
                for (const category of allCategories) {
                    const checkbox = document.createElement("input");
                    checkbox.type = "checkbox";
                    checkbox.id = "edit-category-" + category.id;
                    checkbox.dataset.categoryId = category.id;
                    checkbox.dataset.categoryName = category.name;
                    checkbox.checked = currentIds.has(category.id);

                    const label = document.createElement("label");
                    label.htmlFor = checkbox.id;
                    label.appendChild(document.createTextNode(category.name));

                    const wrapper = document.createElement("span");
                    wrapper.appendChild(checkbox);
                    wrapper.appendChild(label);
                    categoriesDiv.appendChild(wrapper);
                    checkboxes.push(checkbox);
                }
            } catch (error) {
                showMessage("Err: " + error.message, "error");
                console.error(error);
            }

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
            editForm.appendChild(categoriesDiv);
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
                const selectedCategories = checkboxes
                    .filter((cb) => cb.checked)
                    .map((cb) => ({ id: parseInt(cb.dataset.categoryId, 10), name: cb.dataset.categoryName }));
                setButtonLoading(saveButton, true, "Saving...");
                try {
                    const updated = await editPostRequest(post.PostId, newTitle, newContent, selectedCategories);
                    showMessage("Post updated.", "success");
                    displaySinglePost({ ...post, Title: updated.title, Content: updated.content });
                } catch (error) {
                    showMessage("Err: " + error.message, "error");
                    console.error(error);
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
                history.pushState({}, '', '/');
                mainContent.innerHTML = "";
                setActiveNav('all-posts-button');
                getAllPosts();
            } catch (error) {
                showMessage("Err: " + error.message, "error");
                console.error(error);
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

    let commentsListDiv = document.createElement("div");
    commentsListDiv.id = "comments-list";
    commentsSection.appendChild(commentsListDiv);

    let loadMoreButton = document.createElement("button");
    loadMoreButton.type = "button";
    loadMoreButton.className = "btns";
    loadMoreButton.textContent = "Load more comments";
    loadMoreButton.style.display = "none";

    let commentOffset = 0;

    async function loadMoreComments() {
        setButtonLoading(loadMoreButton, true, "Loading...");
        try {
            const { comments, total } = await fetchComments(post.PostId, commentOffset, COMMENTS_PAGE_SIZE);
            if (commentOffset === 0 && total === 0) {
                const emptyMsg = document.createElement("p");
                emptyMsg.className = "empty-state";
                emptyMsg.textContent = "No comments yet — be the first to comment!";
                commentsListDiv.appendChild(emptyMsg);
            }
            comments.forEach(comment => renderComment(comment, commentsListDiv));
            commentOffset += comments.length;
            loadMoreButton.style.display = commentOffset < total ? "inline" : "none";
        } catch (error) {
            showMessage("Err: " + error.message, "error");
            console.error(error);
        } finally {
            setButtonLoading(loadMoreButton, false);
        }
    }

    loadMoreButton.addEventListener("click", loadMoreComments);

    // Fetch the first page of comments for the post
    await loadMoreComments();

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
        event.preventDefault();
        let commentContent = commentInput.value.trim();
        if (commentContent) {
            setButtonLoading(submitButton, true, 'Posting...');
            try {
                const newComment = await submitComment(post.PostId, commentContent);
                commentInput.value = "";
                const existingEmpty = commentsListDiv.querySelector('.empty-state');
                if (existingEmpty) {
                    existingEmpty.remove();
                }
                renderComment(newComment, commentsListDiv);
                commentOffset += 1;
            } catch (error) {
                showMessage("Err: " + error.message, "error");
                console.error(error);
            } finally {
                setButtonLoading(submitButton, false);
            }
        }
    });

    // Add comments section and form to the singlePostDiv
    singlePostDiv.appendChild(commentsSection);
    // Kept outside the scrollable #comments-section (same reasoning as
    // #posts-pagination for the post list): a fixed-height scroll box would
    // bury this control below the fold once comments overflow it.
    singlePostDiv.appendChild(loadMoreButton);
    singlePostDiv.appendChild(commentForm);
}

export async function getPost(id) {
    const response = await fetch(`/getPost?id=${id}`);
    if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to load post (${response.status})`);
    }
    return response.json();
}

async function getPostCategories(postId) {
    const response = await fetch(`/getPostCategories?postId=${postId}`);
    if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to load post categories (${response.status})`);
    }
    return response.json();
}

async function fetchComments(postId, offset = 0, limit = COMMENTS_PAGE_SIZE) {
    const response = await fetch(`/comments?postId=${postId}&limit=${limit}&offset=${offset}`);
    if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to load comments (${response.status})`);
    }
    const total = parseInt(response.headers.get('X-Total-Count') || '0', 10);
    const comments = await response.json();
    return { comments, total };
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

async function editPostRequest(id, title, content, categories = []) {
    const response = await fetch('/editPost', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id, title, content, categories })
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
                    console.error(error);
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
                console.error(error);
                setButtonLoading(deleteButton, false);
            }
        });

        commentElement.appendChild(editButton);
        commentElement.appendChild(deleteButton);
    }

    container.appendChild(commentElement);
}