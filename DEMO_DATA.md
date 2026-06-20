# Demo data reset

## Requirements

- Local or development environment only: `APP_ENV=development` or `APP_ENV=local`
- Configured local PostgreSQL matching the backend `.env`
- Applied migrations, including `000008`
- `DEMO_SEED_PASSWORD` set before reset

## Command

From the repository root:

```bash
DEMO_RESET_CONFIRM=YES DEMO_SEED_PASSWORD='your-password' make demo-reset
```

Windows PowerShell:

```powershell
$env:DEMO_RESET_CONFIRM='YES'
$env:DEMO_SEED_PASSWORD='your-password'
make demo-reset
```

## Safety guards

- The command refuses to run outside `development` / `local`
- The command requires explicit confirmation: `DEMO_RESET_CONFIRM=YES`
- The shared demo password is read only from `DEMO_SEED_PASSWORD`
- The reset deletes all application data in the local database
- Migration history is preserved

## Demo accounts

Password is intentionally not stored in the repository. Use the value passed in `DEMO_SEED_PASSWORD`.

- `anna.smirnova@resourceflow.example` - `admin` - `Administratsiya`
- `mikhail.volkov@resourceflow.example` - `manager` - `Ekspluatatsiya`
- `elena.kuznetsova@resourceflow.example` - `employee` - `Informatsionnye tekhnologii`
- `olga.petrova@resourceflow.example` - `hr` - `Otdel personala`
- `alexey.orlov@resourceflow.example` - `interviewer` - `Prodazhi`
- `igor.sokolov@resourceflow.example` - `employee` - `Informatsionnye tekhnologii` - inactive

## Demo flow

### Admin

- Sign in as `anna.smirnova@resourceflow.example`
- Open `Resources` and verify the warning for `Rabochee mesto A-17`
- Open `Booking Rules` and confirm the disabled rule for `Rabochee mesto`
- Open `Users` and verify the inactive user

### Manager

- Sign in as `mikhail.volkov@resourceflow.example`
- Open `Bookings`
- Approve or reject pending requests
- Complete a confirmed booking if needed

### Employee

- Sign in as `elena.kuznetsova@resourceflow.example`
- Open `Dashboard`, `Resources`, `My Bookings`
- Check busy intervals on resource details
- Create or cancel bookings within seeded availability

## Warning

`make demo-reset` fully deletes application data in the target local database and recreates demo data from scratch.
