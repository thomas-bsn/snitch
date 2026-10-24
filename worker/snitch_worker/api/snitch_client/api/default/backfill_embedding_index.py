from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.backfill_embedding_index_response_202 import BackfillEmbeddingIndexResponse202
from typing import cast



def _get_kwargs(
    *,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/embeddings/backfill",
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> BackfillEmbeddingIndexResponse202 | None:
    if response.status_code == 202:
        response_202 = BackfillEmbeddingIndexResponse202.from_dict(response.json())



        return response_202

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[BackfillEmbeddingIndexResponse202]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Response[BackfillEmbeddingIndexResponse202]:
    """  Ajoute les transcriptions terminées à la file d’indexation sans relancer leur traitement vidéo.

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BackfillEmbeddingIndexResponse202]
     """


    kwargs = _get_kwargs(
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
    origin: str,
    x_csrf_token: str,

) -> BackfillEmbeddingIndexResponse202 | None:
    """  Ajoute les transcriptions terminées à la file d’indexation sans relancer leur traitement vidéo.

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BackfillEmbeddingIndexResponse202
     """


    return sync_detailed(
        client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Response[BackfillEmbeddingIndexResponse202]:
    """  Ajoute les transcriptions terminées à la file d’indexation sans relancer leur traitement vidéo.

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BackfillEmbeddingIndexResponse202]
     """


    kwargs = _get_kwargs(
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
    origin: str,
    x_csrf_token: str,

) -> BackfillEmbeddingIndexResponse202 | None:
    """  Ajoute les transcriptions terminées à la file d’indexation sans relancer leur traitement vidéo.

    Args:
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BackfillEmbeddingIndexResponse202
     """


    return (await asyncio_detailed(
        client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
