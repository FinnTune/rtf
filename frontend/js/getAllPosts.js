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
    }).then((posts) => {
        console.log("PostsAft: ", posts)
        posts.sort((a, b) => (a.CreatedAt > b.CreatedAt) ? -1 : 1);
        let table = document.getElementById('posts-table');
        let tbody = table.querySelector('tbody');
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
        alert("Err: " + error);
        console.log("Err: ", error);
    });
 
    return false;
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
    let comments = await fetchComments(post.PostId);
    console.log("Comments fetch: ", comments)
    comments.forEach(comment => {
        console.log("Comment content: ", comment.content)
        let commentElement = document.createElement("p");
        commentElement.textContent = comment.username + ": " + comment.content;
        commentsSection.appendChild(commentElement);
    });

    // Create a form to submit a new comment
    let commentForm = document.createElement("form");
    let commentInput = document.createElement("input");
    commentInput.type = "text";
    commentInput.name = "comment";
    commentInput.placeholder = "Enter your comment here";
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
            await submitComment(post.PostId, commentContent);
            commentInput.value = "";
            commentsSection.replaceChildren();
            let refreshedHeading = document.createElement("h4");
            refreshedHeading.textContent = "Comments:";
            commentsSection.appendChild(refreshedHeading);
            let updatedComments = await fetchComments(post.PostId);
            updatedComments.forEach(comment => {
                let commentElement = document.createElement("p");
                console.log("Comment content: ", comment.content)
                console.log("Comment username: ", comment.username)
                commentElement.textContent = comment.username + ": " + comment.content;
                commentsSection.appendChild(commentElement);
            });
        }
    });

    // Add comments section and form to the singlePostDiv
    singlePostDiv.appendChild(commentsSection);
    singlePostDiv.appendChild(commentForm);
}

async function fetchComments(postId) {
    const response = await fetch(`/comments?postId=${postId}`);
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
    const result = await response.json();
    return result;
}