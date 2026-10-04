# Event storming (lesson 0.1)

Done on paper first, from "The business in one paragraph" and "Actors" only.
Compared with Chapter 1 afterwards (see the last section).

## Domain events in time order

| # | Event (past tense) | Command | Actor |
|---|---|---|---|
| 1 | VenueRegistered | RegisterVenue | Venue Manager |
| 2 | SectionAdded | AddSection | Venue Manager |
| 3 | VenueActivated | ActivateVenue | Venue Manager |
| 4 | ShowDrafted | DraftShow | Promoter |
| 5 | ShowRescheduled | RescheduleShow | Promoter |
| 6 | ShowPriced | PriceShow | Promoter |
| 7 | ShowPublished | PublishShow | Promoter |
| 8 | InventoryOpened | OpenInventory | policy (on ShowPublished) |
| 9 | SeatsHeld | HoldSeats | Customer |
| 10 | HoldReleased | ReleaseHold | Customer |
| 11 | HoldExpired | ExpireHolds | The Clock |
| 12 | OrderPlaced | PlaceOrder | Customer |
| 13 | PaymentCaptured | Charge | Payment Provider |
| 14 | PaymentFailed | Charge | Payment Provider |
| 15 | OrderPaid | MarkPaid | policy (on PaymentCaptured) |
| 16 | SeatsSold | ConfirmHold | policy (on OrderPaid) |
| 17 | HoldConfirmationFailed | ConfirmHold | policy (on OrderPaid, hold already gone) |
| 18 | TicketsIssued | IssueTickets | policy (on SeatsSold) |
| 19 | TicketsEmailed | SendTicketsEmail | policy (on TicketsIssued) |
| 20 | InventorySoldOut | (raised by SellSeats) | policy |
| 21 | ShowSoldOut | MarkSoldOut | policy (on InventorySoldOut) |
| 22 | VenueRetired | RetireVenue | Venue Manager |
| 23 | ShowCancelled | CancelShow | Promoter, or policy (on VenueRetired) |
| 24 | InventoryClosed | CloseInventory | policy (on ShowCancelled) |
| 25 | TicketsVoided | VoidTickets | policy (on ShowCancelled) |
| 26 | OrderRefunded | RefundOrder | policy (on ShowCancelled / HoldConfirmationFailed) |
| 27 | RefundEmailed | SendRefundEmail | policy (on OrderRefunded) |
| 28 | ShowCompleted | CompleteShow | The Clock |

## Policies ("whenever X, then Y")

- Whenever a **show is published**, open its inventory.
- Whenever a **hold's time runs out**, release its seats.
- Whenever **payment is captured**, mark the order paid; whenever an **order is paid**, confirm the hold.
- Whenever **confirming a hold fails** (it expired while paying), refund the order.
- Whenever **seats are sold**, issue one ticket per seat; whenever **tickets are issued**, email them.
- Whenever the **last seat is sold**, the show is sold out.
- Whenever a **venue retires**, cancel each of its future shows.
- Whenever a **show is cancelled**, close the inventory, void its tickets, refund its orders, email the customers.

## Where the language changes (candidate bounded contexts)

```
Venue Management | Show Scheduling | Ticketing (holds, orders, tickets) | Notifications
                                     ↕ Payment Provider (external)
```

- **Seat**: a physical chair in Venue (section/row/number, accessible?);
  barely exists in Show (Show prices *sections*); a sellable unit with a state
  (Available/Held/Sold) and a price in Ticketing.
- **Event**: the business says "event" for a concert. We say **Show**, and keep
  "event" for domain events.

## Discuss

- *Which event would hurt most if lost?* `SeatsSold` / `TicketsIssued`: a customer
  who paid and got nothing, or a seat sold twice. That points at **Ticketing** as
  the core domain.
- *Is `PaymentCaptured` ours?* No, it's the provider's fact. We translate it into
  our own `OrderPaid`. Payments is a generic subdomain behind a port (ACL).

## Differences from Chapter 1

- I had `ShowRescheduled` and `ShowCompleted`. Chapter 1 doesn't put them on the
  big picture but does model them (SHW-2, SHW-5, the Completed state). Keep them.
- I had `PaymentFailed`. Chapter 1 folds it into the Order lifecycle (TKT-7:
  Pending → PaymentFailed). Same thing, it's a state, not a separate flow.
- Chapter 1 names `HoldConfirmationFailed` only in ADR-010. I found it by asking
  "what if the hold expired while paying?". That's scenario S3.
