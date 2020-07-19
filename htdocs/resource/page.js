window.addEventListener("load", function(ev) {
    console.log("loaded");

    function load_meter_value(el, meter) {
        setInterval(function() {
            var xhdr = new XMLHttpRequest();
            xhdr.addEventListener("load", function(ev) {
                let value = this.responseText;
                console.log(meter, value);
                el.querySelectorAll(".value").forEach(function(elv) {
                    elv.innerText = value;
                });
            });
            xhdr.open("GET", "/api/" + encodeURIComponent(meter) + "/value");
            xhdr.send();

        }, 30 * 1000);
    }

    let sel_meter = document.querySelector("form select[name=\"meter\"");
    sel_meter.addEventListener("change", function(ev) {
        let input = document.querySelector("form input[name=\"value\"");
        let meter = sel_meter.value;
        var xhdr = new XMLHttpRequest();
        xhdr.addEventListener("load", function(ev) {
            let value = this.responseText;
            input.value = value;
            input.parentNode.querySelector(".value").innerText = value;
        });
        xhdr.open("GET", "/api/" + encodeURIComponent(meter) + "/value");
        xhdr.send();
    });

    document.querySelectorAll(".meters .meter").forEach(function(el, k) {
        let meter = el.dataset["meter"];
        load_meter_value(el, meter);
    });
});