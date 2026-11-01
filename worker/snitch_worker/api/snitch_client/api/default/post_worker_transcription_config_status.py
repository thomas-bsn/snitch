from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.post_worker_transcription_config_status_body import PostWorkerTranscriptionConfigStatusBody
from ...models.post_worker_transcription_config_status_response_200 import PostWorkerTranscriptionConfigStatusResponse200
from typing import cast



def _get_kwargs(
    *,
    body: PostWorkerTranscriptionConfigStatusBody,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}






    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/worker/transcription-config/status",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | PostWorkerTranscriptionConfigStatusResponse200:
    if response.status_code == 200:
        response_200 = PostWorkerTranscriptionConfigStatusResponse200.from_dict(response.json())



        return response_200

    if response.status_code == 409:
        response_409 = Error.from_dict(response.json())



        return response_409

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | PostWorkerTranscriptionConfigStatusResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: PostWorkerTranscriptionConfigStatusBody,

) -> Response[Error | PostWorkerTranscriptionConfigStatusResponse200]:
    """  Envoie le résultat du test de configuration depuis le Worker.

    Args:
        body (PostWorkerTranscriptionConfigStatusBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PostWorkerTranscriptionConfigStatusResponse200]
     """


    kwargs = _get_kwargs(
        body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    body: PostWorkerTranscriptionConfigStatusBody,

) -> Error | PostWorkerTranscriptionConfigStatusResponse200 | None:
    """  Envoie le résultat du test de configuration depuis le Worker.

    Args:
        body (PostWorkerTranscriptionConfigStatusBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PostWorkerTranscriptionConfigStatusResponse200
     """


    return sync_detailed(
        client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: PostWorkerTranscriptionConfigStatusBody,

) -> Response[Error | PostWorkerTranscriptionConfigStatusResponse200]:
    """  Envoie le résultat du test de configuration depuis le Worker.

    Args:
        body (PostWorkerTranscriptionConfigStatusBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PostWorkerTranscriptionConfigStatusResponse200]
     """


    kwargs = _get_kwargs(
        body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    body: PostWorkerTranscriptionConfigStatusBody,

) -> Error | PostWorkerTranscriptionConfigStatusResponse200 | None:
    """  Envoie le résultat du test de configuration depuis le Worker.

    Args:
        body (PostWorkerTranscriptionConfigStatusBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PostWorkerTranscriptionConfigStatusResponse200
     """


    return (await asyncio_detailed(
        client=client,
body=body,

    )).parsed
