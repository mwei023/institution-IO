# TODO — Outstanding balances + M-Pesa flow

## Checklist items

- [x] Verify fee/payment schema is correct in finance migrations
- [ ] Build “students with balance > 0” query, scoped by institution_id (backend)
- [ ] Build frontend table: name, class, amount owed, last payment date
- [ ] Test with at least 20 fake students, 3 classes, mixed payment states
- [ ] Connect to M-Pesa sandbox — do a real STK push, confirm it updates balance

## Implementation plan (high level)

1. Backend: add repository method to list outstanding students
   - Inputs: institution_id, optional class_id
   - Outputs: student_id, student_name, class_name, amount_owed, last_payment

2. Backend: add service + handler + route for the listing endpoint

3. Frontend: add API client and a dashboard table that consumes the endpoint

4. Seed: update `scripts/seed_test_data.sql` to generate mixed payment states across 20 students and 3 classes

5. M-Pesa callback: ensure invoice paid_amount/status are updated (or change outstanding calculation to rely on payments)


