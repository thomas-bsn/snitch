from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.embedding_settings import EmbeddingSettings
from ...models.error import Error
from ...models.put_embedding_settings_body import PutEmbeddingSettingsBody
from typing import cast



def _get_kwargs(
    *,
    body: PutEmbeddingSettingsBody,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/admin/embeddings/settings",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> EmbeddingSettings | Error:
    if response.status_code == 200:
        response_200 = EmbeddingSettings.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[EmbeddingSettings | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: PutEmbeddingSettingsBody,
    origin: str,
    x_csrf_token: str,

) -> Response[EmbeddingSettings | Error]:
    """  Enregistre l'adresse et le modèle ; réservé aux Administrateurs. Exige la preuve (validation_token)
    d'un test réussi de cette configuration exacte, à usage unique et de courte durée, rendue par
    testEmbeddingSettings : sinon 409 embedding_validation_required, embedding_validation_expired ou
    embedding_validation_mismatch. Les variables d'environnement restent prioritaires.

    Args:
        origin (str):
        x_csrf_token (str):
        body (PutEmbeddingSettingsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmbeddingSettings | Error]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    body: PutEmbeddingSettingsBody,
    origin: str,
    x_csrf_token: str,

) -> EmbeddingSettings | Error | None:
    """  Enregistre l'adresse et le modèle ; réservé aux Administrateurs. Exige la preuve (validation_token)
    d'un test réussi de cette configuration exacte, à usage unique et de courte durée, rendue par
    testEmbeddingSettings : sinon 409 embedding_validation_required, embedding_validation_expired ou
    embedding_validation_mismatch. Les variables d'environnement restent prioritaires.

    Args:
        origin (str):
        x_csrf_token (str):
        body (PutEmbeddingSettingsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmbeddingSettings | Error
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: PutEmbeddingSettingsBody,
    origin: str,
    x_csrf_token: str,

) -> Response[EmbeddingSettings | Error]:
    """  Enregistre l'adresse et le modèle ; réservé aux Administrateurs. Exige la preuve (validation_token)
    d'un test réussi de cette configuration exacte, à usage unique et de courte durée, rendue par
    testEmbeddingSettings : sinon 409 embedding_validation_required, embedding_validation_expired ou
    embedding_validation_mismatch. Les variables d'environnement restent prioritaires.

    Args:
        origin (str):
        x_csrf_token (str):
        body (PutEmbeddingSettingsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmbeddingSettings | Error]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    body: PutEmbeddingSettingsBody,
    origin: str,
    x_csrf_token: str,

) -> EmbeddingSettings | Error | None:
    """  Enregistre l'adresse et le modèle ; réservé aux Administrateurs. Exige la preuve (validation_token)
    d'un test réussi de cette configuration exacte, à usage unique et de courte durée, rendue par
    testEmbeddingSettings : sinon 409 embedding_validation_required, embedding_validation_expired ou
    embedding_validation_mismatch. Les variables d'environnement restent prioritaires.

    Args:
        origin (str):
        x_csrf_token (str):
        body (PutEmbeddingSettingsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmbeddingSettings | Error
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
