from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_recompute_preview import VoiceRecomputePreview
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/voice-fingerprint-recompute/preview",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceRecomputePreview:
    if response.status_code == 200:
        response_200 = VoiceRecomputePreview.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceRecomputePreview]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | VoiceRecomputePreview]:
    """  Administrateurs uniquement. Ampleur du recalcul des empreintes sur la Parole nette (#395), sans rien
    modifier ni lire de média : empreintes de Locuteur local de production au vecteur d'origine (sans
    mesures du Worker) dans le groupe du modèle d'empreinte actif, dont recalculables, tâches (une par
    diarisation) et clips, sources audio absentes, Locuteurs archivés sans diarisation courante,
    échantillons validés concernés.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceRecomputePreview]
     """


    kwargs = _get_kwargs(

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,

) -> Error | VoiceRecomputePreview | None:
    """  Administrateurs uniquement. Ampleur du recalcul des empreintes sur la Parole nette (#395), sans rien
    modifier ni lire de média : empreintes de Locuteur local de production au vecteur d'origine (sans
    mesures du Worker) dans le groupe du modèle d'empreinte actif, dont recalculables, tâches (une par
    diarisation) et clips, sources audio absentes, Locuteurs archivés sans diarisation courante,
    échantillons validés concernés.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceRecomputePreview
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | VoiceRecomputePreview]:
    """  Administrateurs uniquement. Ampleur du recalcul des empreintes sur la Parole nette (#395), sans rien
    modifier ni lire de média : empreintes de Locuteur local de production au vecteur d'origine (sans
    mesures du Worker) dans le groupe du modèle d'empreinte actif, dont recalculables, tâches (une par
    diarisation) et clips, sources audio absentes, Locuteurs archivés sans diarisation courante,
    échantillons validés concernés.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceRecomputePreview]
     """


    kwargs = _get_kwargs(

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,

) -> Error | VoiceRecomputePreview | None:
    """  Administrateurs uniquement. Ampleur du recalcul des empreintes sur la Parole nette (#395), sans rien
    modifier ni lire de média : empreintes de Locuteur local de production au vecteur d'origine (sans
    mesures du Worker) dans le groupe du modèle d'empreinte actif, dont recalculables, tâches (une par
    diarisation) et clips, sources audio absentes, Locuteurs archivés sans diarisation courante,
    échantillons validés concernés.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceRecomputePreview
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
