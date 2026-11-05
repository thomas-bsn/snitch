from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_model import VoiceModel
from ...models.voice_model_patch import VoiceModelPatch
from typing import cast



def _get_kwargs(
    id: str,
    *,
    body: VoiceModelPatch,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/api/admin/voice-models/{id}".format(id=quote(str(id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceModel:
    if response.status_code == 200:
        response_200 = VoiceModel.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceModel]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: str,
    *,
    client: AuthenticatedClient,
    body: VoiceModelPatch,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceModel]:
    """  Super-administrateur uniquement. Modifie les métadonnées d'une référence (libellé, licence, taille,
    mémoire GPU) ou l'active/la désactive. L'identité (type, dépôt, révision, adaptateur) ne change
    jamais. Refusé (voice_model_in_use) pour désactiver le modèle actif ou le candidat.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (VoiceModelPatch):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModel]
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
    client: AuthenticatedClient,
    body: VoiceModelPatch,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceModel | None:
    """  Super-administrateur uniquement. Modifie les métadonnées d'une référence (libellé, licence, taille,
    mémoire GPU) ou l'active/la désactive. L'identité (type, dépôt, révision, adaptateur) ne change
    jamais. Refusé (voice_model_in_use) pour désactiver le modèle actif ou le candidat.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (VoiceModelPatch):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModel
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
    client: AuthenticatedClient,
    body: VoiceModelPatch,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceModel]:
    """  Super-administrateur uniquement. Modifie les métadonnées d'une référence (libellé, licence, taille,
    mémoire GPU) ou l'active/la désactive. L'identité (type, dépôt, révision, adaptateur) ne change
    jamais. Refusé (voice_model_in_use) pour désactiver le modèle actif ou le candidat.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (VoiceModelPatch):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModel]
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
    client: AuthenticatedClient,
    body: VoiceModelPatch,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceModel | None:
    """  Super-administrateur uniquement. Modifie les métadonnées d'une référence (libellé, licence, taille,
    mémoire GPU) ou l'active/la désactive. L'identité (type, dépôt, révision, adaptateur) ne change
    jamais. Refusé (voice_model_in_use) pour désactiver le modèle actif ou le candidat.

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (VoiceModelPatch):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModel
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
