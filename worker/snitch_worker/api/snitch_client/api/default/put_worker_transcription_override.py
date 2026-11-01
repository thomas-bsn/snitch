from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.put_worker_transcription_override_body import PutWorkerTranscriptionOverrideBody
from ...models.put_worker_transcription_override_response_200 import PutWorkerTranscriptionOverrideResponse200
from typing import cast



def _get_kwargs(
    id: str,
    *,
    body: PutWorkerTranscriptionOverrideBody,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/admin/workers/{id}/transcription/override".format(id=quote(str(id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | PutWorkerTranscriptionOverrideResponse200:
    if response.status_code == 200:
        response_200 = PutWorkerTranscriptionOverrideResponse200.from_dict(response.json())



        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())



        return response_400

    if response.status_code == 404:
        response_404 = Error.from_dict(response.json())



        return response_404

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | PutWorkerTranscriptionOverrideResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    body: PutWorkerTranscriptionOverrideBody,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | PutWorkerTranscriptionOverrideResponse200]:
    """  Administrateurs uniquement. Définit ou supprime l'adresse Whisper propre à ce Worker (vide pour
    supprimer).

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (PutWorkerTranscriptionOverrideBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PutWorkerTranscriptionOverrideResponse200]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    body: PutWorkerTranscriptionOverrideBody,
    origin: str,
    x_csrf_token: str,

) -> Error | PutWorkerTranscriptionOverrideResponse200 | None:
    """  Administrateurs uniquement. Définit ou supprime l'adresse Whisper propre à ce Worker (vide pour
    supprimer).

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (PutWorkerTranscriptionOverrideBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PutWorkerTranscriptionOverrideResponse200
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    body: PutWorkerTranscriptionOverrideBody,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | PutWorkerTranscriptionOverrideResponse200]:
    """  Administrateurs uniquement. Définit ou supprime l'adresse Whisper propre à ce Worker (vide pour
    supprimer).

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (PutWorkerTranscriptionOverrideBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PutWorkerTranscriptionOverrideResponse200]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    body: PutWorkerTranscriptionOverrideBody,
    origin: str,
    x_csrf_token: str,

) -> Error | PutWorkerTranscriptionOverrideResponse200 | None:
    """  Administrateurs uniquement. Définit ou supprime l'adresse Whisper propre à ce Worker (vide pour
    supprimer).

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (PutWorkerTranscriptionOverrideBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PutWorkerTranscriptionOverrideResponse200
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
