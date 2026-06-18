# Wireframe Design Brief — Coffee Mug Shop
*(Paste this whole document into your design tool of choice — Figma Make, v0, Stitch — or hand it to a human designer. It is derived from PRD v1.0 and must not contradict it.)*

---

## The prompt

You are designing wireframes for **Coffee Mug Shop**, an online ordering app for a single coffee shop in Accra, Ghana. Currency is Ghana Cedis (GHS). The frontend will be built in Next.js, so design with standard web components in mind. Produce **low-fidelity wireframes first** — layout, hierarchy, and flow, not visual polish. Two separate surfaces are needed:

1. **Customer storefront** — mobile-first (≈390px), must also work on desktop
2. **Staff/Admin dashboard** — desktop-first (≈1280px), used on a laptop or tablet behind the counter

### Product rules you must respect (from the PRD — do not invent around these)

- Single shop, single menu. No vendor selection, no shop list, no map.
- Fulfilment is **pickup or delivery** with a **flat delivery fee** — no delivery zones, no rider tracking, no map view of a courier.
- Payment happens on **Paystack's hosted page** (customer leaves the app and returns). Design the hand-off and the return, not a card form.
- Order statuses are fixed: `pending_payment → paid → preparing → ready → (out_for_delivery) → completed`, plus `cancelled`. Do not add statuses.
- Checkout requires login. Browsing does not.
- Loyalty points (earn on completed orders, redeem at checkout) exist but are **Phase 2** — design the slots for them (a points badge, a redeem row at checkout) as clearly-marked optional elements.
- All prices display as GHS with two decimals (e.g. GHS 28.00).

### Customer storefront — screens

1. **Home / Menu** — category tabs or chips (Espresso Drinks, Brewed, Pastries…), item cards with image, name, "from GHS X" price, unavailable items visibly disabled. Persistent cart button with item count.
2. **Item detail** — image, description, **variant selector** (e.g. Small / Medium / Large, each with its own price), quantity stepper, "Add to cart — GHS X" button that reflects the selected variant total.
3. **Cart** — line items (name, variant, qty stepper, line total), remove action, subtotal, "Checkout" CTA. Include the **empty-cart state**.
4. **Auth** — login and register (name, email, phone, password). Shown when an unauthenticated user hits checkout. Keep it one column, minimal.
5. **Checkout** — fulfilment toggle (Pickup / Delivery). Delivery reveals address (free-text) and phone fields and adds a flat fee row. Order summary: subtotal, delivery fee, (Phase 2: points redemption row with a toggle/input), total. CTA: "Pay GHS X with Paystack". Make it explicit the user will be redirected to pay.
6. **Payment return / Order tracking** — single screen. While the webhook confirms: a "Confirming your payment…" state. Then a **status stepper/timeline** (Paid → Preparing → Ready → Out for delivery → Completed) that updates live without refresh. Show order number, items, total, fulfilment details. Include a **payment failed** state with a retry path.
7. **Order history** — list of past orders (date, total, status chip), tapping opens the tracking/detail view above.
8. **Profile** — name, contact, logout. (Phase 2: loyalty balance and points ledger list.)

### Staff/Admin dashboard — screens

1. **Login** — same auth, no register link.
2. **Order queue** — the core staff screen. Columns or filter tabs by status (New/Paid, Preparing, Ready, Out for delivery). Each order card: order number, time since paid, pickup-or-delivery badge, item count, total. Primary action on each card advances the order to its next status (e.g. "Start preparing", "Mark ready"). New paid orders must be visually loud — this screen is glanced at, not read.
3. **Order detail** — full line items, customer name + phone, delivery address if applicable, status history timeline (who changed what, when), cancel action (only valid from early statuses).
4. **Menu management** — table/list of items grouped by category with an inline **availability toggle**, edit and delete actions, "Add item" CTA.
5. **Item editor** — name, description, category select, image URL field, and a repeatable **variants block** (variant name + price) with add/remove.
6. **(Phase 2) Reports** — two simple cards/charts: orders per day, revenue per day.

### Required flows to diagram

- Browse → item → cart → checkout → Paystack redirect → return → live tracking
- Staff: new paid order appears → advance through statuses → customer screen reflects each change
- Unavailable item: what the customer sees, what the admin toggle looks like

### States that must exist in the wireframes

Empty cart · empty order history · payment confirming · payment failed · order cancelled · item unavailable · staff queue with zero new orders.

### Tone (for later hi-fi, not the wireframes)

Warm, simple, café-feeling — not corporate dashboard energy on the customer side. Admin side: dense, fast, utilitarian.

### Deliverables

1. Low-fi wireframes for every screen above, mobile frames for customer, desktop frames for admin
2. A one-page user-flow diagram covering the three flows listed
3. A short component inventory (buttons, cards, status chips, stepper) so the Next.js build maps 1:1 to the designs
