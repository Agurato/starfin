function request(el) {
  let imdbId = el.getAttribute("tmdbId");
  let mediaType = el.getAttribute("mediaType");
  let action = el.getAttribute("action");
  let url = "/" + mediaType.toLowerCase() + "/" + imdbId + "/" + action;

  fetch(url, {
    method: "POST",
  }).then((res) => {
    if (res.status == 200) {
      res.json().then((data) => {
        console.log(data.message);
        switch (action) {
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
    } else {
      res.json().then((data) => {
        console.error(res.status, data.message);
      });
    }
  });
}

function deleteVolume(el) {
  let volumeId = el.getAttribute("volumeId");
  let url = "/admin/deletevolume";

  fetch(url, {
    method: "POST",
    body: new URLSearchParams({ "volumeId": volumeId }),
  }).then((res) => {
    if (res.status == 200) {
      res.json().then((data) => {
        if (data.error) {
          console.error(data.error);
        } else {
          console.log(data.message);
          el.parentNode.parentNode.parentNode.removeChild(
            el.parentNode.parentNode,
          );
        }
      });
    } else {
      res.json().then((data) => {
        console.error(res.status, data.message);
      });
    }
  });
}

function deleteUser(el) {
  let userId = el.getAttribute("userId");
  let url = "/admin/deleteuser";

  fetch(url, {
    method: "POST",
    body: new URLSearchParams({ "userId": userId }),
  }).then((res) => {
    if (res.status == 200) {
      res.json().then((data) => {
        if (data.error) {
          console.error(data.error);
        } else {
          console.log(data.message);
          el.parentNode.parentNode.parentNode.removeChild(
            el.parentNode.parentNode,
          );
        }
      });
    } else {
      res.json().then((data) => {
        console.error(res.status, data.message);
      });
    }
  });
}

function retFalse() {
  return false;
}
