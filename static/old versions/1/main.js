
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

(function main() {
	set_on_scroll();
	graph(fast_out_slow_in);
	graph((t) => 0.9 - fast_out_slow_in(t) * 0.8);
	graph((t) => 0.5 + Math.cos(Math.PI + t * Math.PI) * 0.3);
	graph((t) => 0.5 + Math.cos(t * Math.PI) * 0.2);
})();

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
		dot.style.background = `hsl(${i*256}, 60%, 70%)`;
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
	//{query: "nav", y_level: 0},
	//{ query: "#hero", y_level: 25 },
]
function parallax_scroll() {
	const y = window.scrollY;
	for (let trig of scroll_triggers) {
		const element = document.querySelector(trig.query);
		let func = "remove";
		if (y > trig.y_level)
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
	}

	const title = document.querySelector("#hero h1");
	const top_value = `${40 * (end - percentage * 0.3) / end}vh`
	title.style.top = top_value;
	title.style.transform = `scale(${(end - percentage * 0.5) / end})`;

	const main = document.querySelector("main");
	main.style.paddingTop = `${100 * (end - percentage * 0.45) / end}vh`;
}