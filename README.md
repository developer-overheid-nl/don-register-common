# DON Register Common

Gedeelde Go-packages voor de DON register-services.

Deze module bevat infrastructuurcode die zowel door `don-api-register` als
`don-oss-register` gebruikt wordt. De domeinlogica blijft in de afzonderlijke
registers; deze repository bevat alleen generieke helpers en kleine
bouwstenen.

## Packages

- `database`: Postgres/GORM connectie helper.
- `filters`: gedeelde filter response-modellen en option builders.
- `httpclient`: HTTP helpers en TOOI organisatielabel lookup.
- `pagination`: pagination model, normalisatie en HTTP response headers.
- `problem`: Problem Details response types.
- `query`: query helpers zoals SQL LIKE escaping en filter counts.
- `router`: Gin/CORS/API-version bootstrap.
- `typesense`: Typesense configuratie, base document en upsert helper.

## Gebruik

Voeg de module toe aan een register-service:

```bash
go get github.com/developer-overheid-nl/don-register-common
```

Gebruik vanuit register-repos bij voorkeur kleine wrappers of type aliases als
de bestaande package-API stabiel moet blijven.

## Ontwikkelen

Voer de tests uit met:

```bash
go test ./...
```

Bij wijzigingen in deze module moeten de afhankelijke register-repos hun
`go.mod` bijwerken naar de nieuwe pseudo-version of tag.
