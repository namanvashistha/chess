console.log("Hello, World!");

// function to fetch the user anonymous token from the local storage and if not found, create a new one from server
function fetchUserToken() {
    let userToken = localStorage.getItem('userToken');
    if (!userToken) {
        fetch('/api/user', { method: 'POST' })
            .then(res => res.json())
            .then(data => {
                if (data.response_key === "SUCCESS") {
                    userToken = data.data.token;
                    localStorage.setItem('userToken', userToken);
                } else {
                    fetch('/api/user', { method: 'POST' })
                    .then(res => res.json())
                    .then(data => {
                        if (data.response_key === "SUCCESS") {
                            userToken = data.data.token;
                            localStorage.setItem('userToken', userToken);
                        } else {
                            console.error('Error fetching user token:', data);
                        }
                    })
                    .catch(err => console.error('Error fetching user token:', err));
                }
            })
            .catch(err => console.error('Error fetching user token:', err));
    }
    console.log("userToken2:", userToken);
    return userToken;
}

// Call the function to fetch the user token when the page is loaded
fetchUserToken();