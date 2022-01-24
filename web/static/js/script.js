function request(el) {
    let imdbId = el.getAttribute("tmdbId");
    let mediaType = el.getAttribute("mediaType");
    let action = el.getAttribute("action");
    url = "/" + mediaType.toLowerCase() + "/" + imdbId + "/" + action;

    fetch(url, {
        method: "POST",
    }).then(res => {
        if(res.status == 200) {
            res.json().then(data => {
                console.log(data.message)
                switch(action) {
                    case "add":
                        el.innerHTML = "Un-request";
                        el.setAttribute("action", "remove");
                        break;
                    case "remove":
                        el.innerHTML = "Request";
                        el.setAttribute("action", "add");
                        break;
                }
            });
        }
        else {
            res.json().then(data => {
                console.error(res.status, data.message);
            })
        }
    });
}