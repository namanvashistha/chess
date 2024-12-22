console.log("Hello, World!");

// function to fetch the user anonymous token from the local storage and if not found, create a new one from server

function formatUserName(name) {
    if (!name) {
        return "Anonymous";
    }
    return name.split("-").map(word => word.charAt(0).toUpperCase() + word.slice(1)).slice(0, 2).join(" ");
}
function fetchUserToken() {
    userName = "";
    let userToken = localStorage.getItem('userToken');
    if (!userToken) {
        fetch('/api/user', { method: 'POST' })
            .then(res => res.json())
            .then(data => {
                if (data.response_key === "SUCCESS") {
                    userToken = data.data.token;
                    localStorage.setItem('userToken', userToken);
                    localStorage.setItem('userData', JSON.stringify(data.data));
                } else {
                    fetch('/api/user', { method: 'POST' })
                        .then(res => res.json())
                        .then(data => {
                            if (data.response_key === "SUCCESS") {
                                userToken = data.data.token;
                                localStorage.setItem('userToken', userToken);
                                localStorage.setItem('userData', JSON.stringify(data.data));
                            } else {
                                console.error('Error fetching user token:', data);
                            }
                        })
                        .catch(err => console.error('Error fetching user token:', err));
                }
            })
            .catch(err => console.error('Error fetching user token:', err));
    }
    userData = localStorage.getItem('userData');
    if (!userData) {
        fetch('/api/user/me', {
            method: 'POST',
            body: JSON.stringify({ token: userToken }),
            headers: {
                'Content-Type': 'application/json',
            },
        })
            .then(res => res.json())
            .then(data => {
                if (data.response_key === "SUCCESS") {
                    const user = data.data;
                    userName = user.name;;
                    //convert data.data to string before storing in local storage
                    localStorage.setItem('userData', JSON.stringify(data.data));
                } else {
                    console.error('Error fetching user:', data);
                }
            })
            .catch(err => console.error('Error fetching user:', err));
    }
    else {
        const user = JSON.parse(userData);
        userName = user.name;
    }
    console.log("User Name", userName);
    if (document.getElementById('user-name') && userName) {
        document.getElementById('user-name').textContent = formatUserName(userName);
    }
    return userToken;
}

// Call the function to fetch the user token when the page is loaded
fetchUserToken();