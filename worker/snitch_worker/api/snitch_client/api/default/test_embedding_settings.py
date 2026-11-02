from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.embedding_validation import EmbeddingValidation
from ...models.error import Error
from ...models.test_embedding_settings_body import TestEmbeddingSettingsBody
from typing import cast



def _get_kwargs(
    *,
    body: TestEmbeddingSettingsBody,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/embeddings/settings/test",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> EmbeddingValidation | Error:
    if response.status_code == 200:
        response_200 = EmbeddingValidation.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[EmbeddingValidation | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: TestEmbeddingSettingsBody,
    origin: str,
    x_csrf_token: str,

) -> Response[EmbeddingValidation | Error]:
    """  Teste l'adresse et le modèle contre le vrai service, sans rien enregistrer ni toucher à l'index ;
    réservé aux Administrateurs. En cas de succès, rend une preuve de validation liée aux valeurs
    effectives et à l'utilisateur, valable quelques minutes et à usage unique.

    Args:
        origin (str):
        x_csrf_token (str):
        body (TestEmbeddingSettingsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmbeddingValidation | Error]
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
    body: TestEmbeddingSettingsBody,
    origin: str,
    x_csrf_token: str,

) -> EmbeddingValidation | Error | None:
    """  Teste l'adresse et le modèle contre le vrai service, sans rien enregistrer ni toucher à l'index ;
    réservé aux Administrateurs. En cas de succès, rend une preuve de validation liée aux valeurs
    effectives et à l'utilisateur, valable quelques minutes et à usage unique.

    Args:
        origin (str):
        x_csrf_token (str):
        body (TestEmbeddingSettingsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmbeddingValidation | Error
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
    body: TestEmbeddingSettingsBody,
    origin: str,
    x_csrf_token: str,

) -> Response[EmbeddingValidation | Error]:
    """  Teste l'adresse et le modèle contre le vrai service, sans rien enregistrer ni toucher à l'index ;
    réservé aux Administrateurs. En cas de succès, rend une preuve de validation liée aux valeurs
    effectives et à l'utilisateur, valable quelques minutes et à usage unique.

    Args:
        origin (str):
        x_csrf_token (str):
        body (TestEmbeddingSettingsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmbeddingValidation | Error]
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
    body: TestEmbeddingSettingsBody,
    origin: str,
    x_csrf_token: str,

) -> EmbeddingValidation | Error | None:
    """  Teste l'adresse et le modèle contre le vrai service, sans rien enregistrer ni toucher à l'index ;
    réservé aux Administrateurs. En cas de succès, rend une preuve de validation liée aux valeurs
    effectives et à l'utilisateur, valable quelques minutes et à usage unique.

    Args:
        origin (str):
        x_csrf_token (str):
        body (TestEmbeddingSettingsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmbeddingValidation | Error
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
