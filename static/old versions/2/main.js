
// Control point
class CP {
	x = 0;
	y = 0;

	constructor(new_x, new_y) {
		this.x = new_x;
		this.y = new_y;
	}

	times(scalar) {
		return new CP(this.x * scalar, this.y * scalar);
	}

	plus(other) {
		return new CP(this.x + other.x, this.y + other.y);
	}
}

let passive_supported = false;
try {
	const options = {
		get passive() {
			passive_supported = true;
			return false;
		}
	};
	window.addEventListener("test", null, options);
	window.removeEventListener("test", null, options);
} catch (err) {
	passive_supported = false;
}

function graph(func) {
	const dot_container = document.createElement("div");
	dot_container.classList.add("dot-container");
	document.body.prepend(dot_container);
	dot_container.setAttribute("function", func.toString());
	for (let i = 0; i < 1; i += 0.01) {
		if (Math.abs(i - 0.5) < 0.12)
			continue;
		let dot = document.createElement("div");
		dot.classList.add("graph-dot");
		dot.style.left = (i * 100) + "%";
		dot.style.top = (100 - func(i) * 100) + "vh";
		dot.style.background = `hsl(${i * 256}, 60%, 70%)`;
		dot_container.appendChild(dot);
	}
}

function set_on_scroll() {
	if (passive_supported)
		document.addEventListener("scroll", () => {
			parallax_scroll();
		}, { passive: true });
	else
		document.addEventListener("scroll", () => {
			parallax_scroll();
		});
}

let scroll_triggers = [
	//{ query: "nav", relative: true, y_level: 55 },
]
function parallax_scroll() {
	const y = window.scrollY;
	const rel_y = window.scrollY / window.innerHeight * 100;
	for (let trig of scroll_triggers) {
		const element = document.querySelector(trig.query);
		let func = "remove";
		if ((trig.relative ? rel_y : y) > trig.y_level)
			func = "add";
		element.classList[func]("scrolled");
	}
	title_scroll(y);
	//nav_scroll(y);
}

/*  Returns the x value when y = t on the cubic bezier curve defined by
	the p1x, p1y, p2x, and p2y parameters */
function cubic_bezier(p1x, p1y, p2x, p2y, t) {
	if (t > 1)
		t = 1;

	// Not needed because always zero
	//const p0 = new CP(1, 1);
	const p1 = new CP(p1x, p1y)
	const p2 = new CP(p2x, p2y);
	const p3 = new CP(1, 1);

	return p1.times(3 * t * (1 - t) * (1 - t))
		.plus(p2.times(3 * t * t * (1 - t)))
		.plus(p3.times(t * t * t)).y;
}

function fast_out_slow_in(t) {
	return cubic_bezier(0.4, 0, 0.2, 1, t);
}

let title_scroll_been_above = false;
function title_scroll(y) {
	const end = 0.8;
	let percentage = fast_out_slow_in(y / window.innerHeight) / end;

	if (percentage > end) {
		if (title_scroll_been_above)
			return;
		else
			percentage = end;
	} else {
		title_scroll_been_above = false;
	}

	const title = document.querySelector("#hero h1");
	const top_value = `${40 * (end - percentage * 0.3) / end}vh`
	title.style.top = top_value;
	title.style.transform = `scale(${(end - percentage * 0.5) / end})`;

	const main = document.querySelector("main");
	main.style.paddingTop = `${100 * (end - percentage * 0.45) / end}vh`;
}

function ripple_size(height, width) {
	return (Math.max(height, width) * Math.sqrt(2)) / 16;
}

function relative_click_coordinates(event, target) {
	let rect = target.getBoundingClientRect();
	return {
		x: event.pageX - rect.left,
		y: event.pageY - rect.top
	};
}

function applyStylesToRipple(element, event, target) {
	const parent_height = target.clientHeight,
		parent_width = target.clientWidth,
		size =_size(
			target.clientHeight,
			target.clientWidth
		);

	const coords = relative_click_coordinates(event, target);
	element.style.top = coords.y - 8 + "px";
	element.style.left = coords.x - 8 + "px";
	element.style.transition = "500ms cubic-bezier(0.4, 0, 0.2, 1)";
	requestAnimationFrame(() => {
		element.style.transform = "scale(" + size + ")";
		element.style.top = parent_height / 2 - 8 + "px";
		element.style.left = parent_width / 2 - 8 + "px";
	}, 10);
}

function ripple_cleanup(ripple) {
	ripple.setAttribute("removal-scheduled", "f");
	window.setTimeout(function () {
		ripple.style.transition = "1200ms cubic-bezier(0, 0, 0.2, 1)";
		ripple.style.opacity = 0;
		ripple.style.width = "32px";
		ripple.style.height = "32px";
		window.setTimeout(function () {
			if (ripple)
				ripple.remove();
		}, 750);
	}, 120);
}

function create_ripple(parent) {
	const ripple = document.createElement("div");
	ripple.classList.add("ink-splash");
	return parent.appendChild(ripple);
}

function ripple_event_handler(event) {
	const target = event.currentTarget;
	requestAnimationFrame(() => {
		if (rippling_allowed) {
			rippling_allowed = false;

			if (!target.classList.contains("disabled")) {
				const_element = create_ripple(target);
				applyStylesToRipple(ripple_element, event, target);
			}
			window.setTimeout(function () {
				rippling_allowed = true;
			}, 50);
		}
	});
}

function init_ripple() {
	/* 	For some reason thes cause other event listeners
	 *	to not fire on pale moon. I'm including Goanna and Gecko
	 *	too since I don't know where the issue lies and it's
	 *	better to be safe than sorry.
	 */
	if (navigator.userAgent.includes("PaleMoon") ||
		navigator.userAgent.includes("Goanna"))
		return;

	const_containers = document.getElementsByClassName("ripple")

	// Apply event listener to each of theContainers
	for (let container of ripple_containers) {
		if (container.hasAttribute("ripple-status"))
			continue;

		container.setAttribute("ripple-status", "activated");
		container.addEventListener("mousedown", event => {
			ripple_event_handler(event);
		});
		if (passive_supported) {
			container.addEventListener("touchstart", event => {
				ripple_event_handler(event);
			}, { passive: true });
		}
		else {
			container.addEventListener("touchstart", event => {
				ripple_event_handler(event);
			});
		}

		// Delete if the mouse leaves the container
		container.addEventListener("mouseleave", event => {
			delete_all_ripples();
		});

		// Clean ups if mouseup or touchend occurs outside container
		document.addEventListener("mouseup", event => {
			delete_all_ripples();
		});

		document.addEventListener("touchend", event => {
			delete_all_ripples();
		});
	}
}

// To stop multiple events from firing
let rippling_allowed = true;
let deleting_ripples_allowed = true;

function delete_all_ripples() {
	if (deleting_ripples_allowed) {
		deleting_ripples_allowed = false;
		const targets = document.getElementsByClassName("ink-splash");
		for (let target of targets) {
			if (!target.hasAttribute("removal-scheduled")) {
				ripple_cleanup(target);
			}
		}
		window.setTimeout(() => {
			deleting_ripples_allowed = true;
		}, 50);
	}
}

(function main() {
	set_on_scroll();
	init_ripple();
	graph(fast_out_slow_in);
	graph((t) => 0.9 - fast_out_slow_in(t) * 0.8);
	graph((t) => 0.5 + Math.cos(Math.PI + t * Math.PI) * 0.3);
	graph((t) => 0.5 + Math.cos(t * Math.PI) * 0.2);
})();
